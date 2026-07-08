package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/scc-operator/internal/consts"
	v1 "github.com/rancher/scc-operator/pkg/apis/scc.cattle.io/v1"
)

func TestHasSccManagedByLabel(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		input    runtimeMetaObject
		expected bool
	}{
		{
			name:     "no labels",
			input:    &v1.Registration{},
			expected: false,
		},
		{
			name: "has SCC managed-by label",
			input: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelSccManagedBy: "test-manager",
					},
				},
			},
			expected: true,
		},
		{
			name: "has both labels",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelSccManagedBy: "test-manager",
						consts.LabelK8sManagedBy: "Helm",
					},
				},
			},
			expected: true,
		},
		{
			name: "only has k8s managed-by label",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "Helm",
					},
				},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := HasSccManagedByLabel(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetSccManagedByValue(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		input    runtimeMetaObject
		expected string
	}{
		{
			name:     "no labels",
			input:    &v1.Registration{},
			expected: "",
		},
		{
			name: "has SCC managed-by label",
			input: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelSccManagedBy: "test-manager",
					},
				},
			},
			expected: "test-manager",
		},
		{
			name: "has both labels",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelSccManagedBy: "rancher-scc-operator",
						consts.LabelK8sManagedBy: "Helm",
					},
				},
			},
			expected: "rancher-scc-operator",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := GetSccManagedByValue(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestShouldManageByScc(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		manager  string
		input    runtimeMetaObject
		expected bool
	}{
		{
			name:     "no labels - unmanaged",
			manager:  "test-manager",
			input:    &v1.Registration{},
			expected: false,
		},
		{
			name:    "SCC label matches",
			manager: "test-manager",
			input: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelSccManagedBy: "test-manager",
					},
				},
			},
			expected: true,
		},
		{
			name:    "SCC label does not match",
			manager: "different-manager",
			input: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelSccManagedBy: "test-manager",
					},
				},
			},
			expected: false,
		},
		{
			name:    "SCC label takes precedence over k8s label",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelSccManagedBy: "test-manager",
						consts.LabelK8sManagedBy: "Helm",
					},
				},
			},
			expected: true,
		},
		{
			name:    "Falls back to k8s label when SCC label absent",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "test-manager",
					},
				},
			},
			expected: true,
		},
		{
			name:    "Falls back to k8s label when SCC manager has _secret-broker suffix",
			manager: "test-manager_secret-broker",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "test-manager",
					},
				},
			},
			expected: true,
		},
		{
			name:    "Helm label is treated as manageable",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "Helm",
					},
				},
			},
			expected: true,
		},
		{
			name:    "k8s label does not match and is not Helm",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "some-other-tool",
					},
				},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := ShouldManageByScc(tc.input, tc.manager)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTakeSccOwnership(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name           string
		manager        string
		input          runtimeMetaObject
		expectedLabels map[string]string
	}{
		{
			name:    "no labels - adds SCC label only",
			manager: "test-manager",
			input:   &v1.Registration{},
			expectedLabels: map[string]string{
				consts.LabelSccManagedBy: "test-manager",
			},
		},
		{
			name:    "preserves existing k8s managed-by label",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "Helm",
					},
				},
			},
			expectedLabels: map[string]string{
				consts.LabelK8sManagedBy: "Helm",
				consts.LabelSccManagedBy: "test-manager",
			},
		},
		{
			name:    "overwrites existing SCC label",
			manager: "new-manager",
			input: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelSccManagedBy: "old-manager",
						consts.LabelK8sManagedBy: "Helm",
					},
				},
			},
			expectedLabels: map[string]string{
				consts.LabelSccManagedBy: "new-manager",
				consts.LabelK8sManagedBy: "Helm",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := TakeSccOwnership(tc.input, tc.manager)
			assert.Equal(t, tc.expectedLabels, result.GetLabels())
		})
	}
}

func TestTakeFullOwnership(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name           string
		manager        string
		input          runtimeMetaObject
		expectedLabels map[string]string
	}{
		{
			name:    "no labels - adds both labels",
			manager: "test-manager",
			input:   &v1.Registration{},
			expectedLabels: map[string]string{
				consts.LabelSccManagedBy: "test-manager",
				consts.LabelK8sManagedBy: "test-manager",
			},
		},
		{
			name:    "overwrites existing k8s managed-by label",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "Helm",
					},
				},
			},
			expectedLabels: map[string]string{
				consts.LabelK8sManagedBy: "test-manager",
				consts.LabelSccManagedBy: "test-manager",
			},
		},
		{
			name:    "overwrites both labels",
			manager: "new-manager",
			input: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelSccManagedBy: "old-manager",
						consts.LabelK8sManagedBy: "Helm",
					},
				},
			},
			expectedLabels: map[string]string{
				consts.LabelSccManagedBy: "new-manager",
				consts.LabelK8sManagedBy: "new-manager",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := TakeFullOwnership(tc.input, tc.manager)
			assert.Equal(t, tc.expectedLabels, result.GetLabels())
		})
	}
}
