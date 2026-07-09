package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/rancher/scc-operator/internal/consts"
	v1 "github.com/rancher/scc-operator/pkg/apis/scc.cattle.io/v1"
)

// RuntimeMetaObject is an interface for a K8s Object to be used with a specific controller.
type runtimeMetaObject interface {
	runtime.Object
	metav1.Object
}

func TestHasManagedByLabel(t *testing.T) {
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
			name: "registration with hasManagedBy label",
			input: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "test-manager",
						consts.LabelSccManagedBy: "test-value",
					},
				},
			},
			expected: true,
		},
		{
			name: "secret with hasManagedBy label",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelSccManagedBy: "test-value",
						consts.LabelK8sManagedBy: "test-manager",
					},
				},
			},
			expected: true,
		},
		{
			name:     "configmap without label",
			input:    &corev1.ConfigMap{},
			expected: false,
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable for parallel execution
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := HasManagedByLabel(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetManagedByValue(t *testing.T) {
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
			name: "registration with hasManagedBy label",
			input: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "test-manager",
						consts.LabelSccManagedBy: "test-value",
					},
				},
			},
			expected: "test-manager",
		},
		{
			name: "registration with hasManagedBy label but different manager",
			input: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "different-test-manager",
						consts.LabelSccManagedBy: "test-value",
					},
				},
			},
			expected: "different-test-manager",
		},
		{
			name: "secret with hasManagedBy label",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelSccManagedBy: "test-value",
						consts.LabelK8sManagedBy: "test-manager",
					},
				},
			},
			expected: "test-manager",
		},
		{
			name:     "configmap without label",
			input:    &corev1.ConfigMap{},
			expected: "",
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable for parallel execution
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := GetManagedByValue(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestShouldManage(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		manager  string
		input    runtimeMetaObject
		expected bool
	}{
		{
			name:     "no labels",
			manager:  "test-manager",
			input:    &v1.Registration{},
			expected: false,
		},
		{
			name:    "registration with both labels matching",
			manager: "test-manager",
			input: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "test-manager",
						consts.LabelSccManagedBy: "test-manager_secret-broker",
					},
				},
			},
			expected: true,
		},
		{
			name:    "registration with hasManagedBy label but different manager",
			manager: "different-test-manager",
			input: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "test-manager",
						consts.LabelSccManagedBy: "test-value",
					},
				},
			},
			expected: false,
		},
		{
			name:    "secret with both labels matching",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelSccManagedBy: "test-manager_secret-broker",
						consts.LabelK8sManagedBy: "test-manager",
					},
				},
			},
			expected: true,
		},
		{
			name:     "configmap without label",
			manager:  "test-manager",
			input:    &corev1.ConfigMap{},
			expected: false,
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable for parallel execution
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := ShouldManage(tc.input, tc.manager)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestShouldAdopt(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		manager  string
		input    runtimeMetaObject
		expected bool
	}{
		{
			name:     "no labels at all - should adopt",
			manager:  "test-manager",
			input:    &v1.Registration{},
			expected: false, // No labels means nothing to match against
		},
		{
			name:    "k8s label only - matching operator - should adopt",
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
			name:    "k8s label only - Helm - should adopt",
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
			name:    "k8s label only - different operator - should NOT adopt",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "argocd",
					},
				},
			},
			expected: false,
		},
		{
			name:    "SCC label only - matching - should adopt (fallback case)",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelSccManagedBy: "test-manager_secret-broker",
					},
				},
			},
			expected: true,
		},
		{
			name:    "SCC label only - not matching - should NOT adopt",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelSccManagedBy: "other-operator_secret-broker",
					},
				},
			},
			expected: false,
		},
		{
			name:    "both labels - matching operator - should adopt",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "test-manager",
						consts.LabelSccManagedBy: "test-manager_secret-broker",
					},
				},
			},
			expected: true,
		},
		{
			name:    "both labels - Helm k8s + matching SCC - should adopt",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "Helm",
						consts.LabelSccManagedBy: "test-manager_secret-broker",
					},
				},
			},
			expected: true,
		},
		{
			name:    "both labels - matching k8s but wrong SCC format - should NOT adopt",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "test-manager",
						consts.LabelSccManagedBy: "wrong-format",
					},
				},
			},
			expected: false,
		},
		{
			name:    "both labels - Helm k8s but wrong SCC - should NOT adopt",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "Helm",
						consts.LabelSccManagedBy: "other-operator_secret-broker",
					},
				},
			},
			expected: false,
		},
		{
			name:    "both labels - different k8s operator + different SCC - should NOT adopt",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "argocd",
						consts.LabelSccManagedBy: "other-operator_secret-broker",
					},
				},
			},
			expected: false,
		},
		{
			name:    "both labels - different k8s operator but matching SCC - should NOT adopt",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "argocd",
						consts.LabelSccManagedBy: "test-manager_secret-broker",
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
			result := ShouldAdopt(tc.input, tc.manager)
			assert.Equal(t, tc.expected, result, "ShouldAdopt result mismatch")
		})
	}
}

func TestShouldManage_WithSccOnlyFallback(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		manager  string
		input    runtimeMetaObject
		expected bool
	}{
		{
			name:    "SCC label only - matching - should manage (fallback case)",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelSccManagedBy: "test-manager_secret-broker",
					},
				},
			},
			expected: true,
		},
		{
			name:    "SCC label only - not matching - should NOT manage",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelSccManagedBy: "other-operator_secret-broker",
					},
				},
			},
			expected: false,
		},
		{
			name:    "SCC label only - wrong format - should NOT manage",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelSccManagedBy: "test-manager", // Missing _secret-broker suffix
					},
				},
			},
			expected: false,
		},
		{
			name:    "both labels - k8s Helm + matching SCC - should manage",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "Helm",
						consts.LabelSccManagedBy: "test-manager_secret-broker",
					},
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := ShouldManage(tc.input, tc.manager)
			assert.Equal(t, tc.expected, result, "ShouldManage result mismatch")
		})
	}
}

func TestTakeOwnership(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name                 string
		manager              string
		input                runtimeMetaObject
		expectedShouldManage bool
		expectedLabels       map[string]string
	}{
		{
			name:                 "no labels",
			manager:              "test-manager",
			input:                &v1.Registration{},
			expectedShouldManage: false,
			expectedLabels: map[string]string{
				consts.LabelK8sManagedBy: "test-manager",
				consts.LabelSccManagedBy: "test-manager_secret-broker",
			},
		},
		{
			name:    "registration with both labels matching",
			manager: "test-manager",
			input: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "test-manager",
						consts.LabelSccManagedBy: "test-manager_secret-broker",
					},
				},
			},
			expectedShouldManage: true, // Both labels match
			expectedLabels: map[string]string{
				consts.LabelK8sManagedBy: "test-manager",
				consts.LabelSccManagedBy: "test-manager_secret-broker",
			},
		},
		{
			name:    "registration with hasManagedBy label but different manager",
			manager: "different-test-manager",
			input: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "test-manager",
						consts.LabelSccManagedBy: "test-value",
					},
				},
			},
			expectedShouldManage: false,
			expectedLabels: map[string]string{
				consts.LabelK8sManagedBy: "different-test-manager",
				consts.LabelSccManagedBy: "different-test-manager_secret-broker", // TakeOwnership overwrites both
			},
		},
		{
			name:    "secret with both labels matching",
			manager: "test-manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelSccManagedBy: "test-manager_secret-broker",
						consts.LabelK8sManagedBy: "test-manager",
					},
				},
			},
			expectedShouldManage: true, // Both labels match
			expectedLabels: map[string]string{
				consts.LabelSccManagedBy: "test-manager_secret-broker",
				consts.LabelK8sManagedBy: "test-manager",
			},
		},
		{
			name:                 "configmap without label",
			manager:              "test-manager",
			input:                &corev1.ConfigMap{},
			expectedShouldManage: false,
			expectedLabels: map[string]string{
				consts.LabelK8sManagedBy: "test-manager",
				consts.LabelSccManagedBy: "test-manager_secret-broker",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable for parallel execution
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Check if the object should be managed before taking ownership
			shouldManage := ShouldManage(tc.input, tc.manager)
			assert.Equal(t, tc.expectedShouldManage, shouldManage)
			if shouldManage {
				// If the object already had the correct owner, it should have been managed before
				assert.Equal(t, tc.manager, tc.input.GetLabels()[consts.LabelK8sManagedBy],
					"Object should have had the correct owner before taking ownership")
				assert.Equal(t, tc.expectedLabels, tc.input.GetLabels())

				// Take ownership of the -already owned- object
				result := TakeOwnership(tc.input, tc.manager)

				// Reverify the results are the same
				assert.Equal(t, tc.manager, result.GetLabels()[consts.LabelK8sManagedBy],
					"Object should have had the correct owner before taking ownership")
				assert.Equal(t, tc.expectedLabels, result.GetLabels())
			} else {
				assert.NotEqual(t, tc.expectedLabels, tc.input.GetLabels(),
					"Object should not have expected labels until after TakeOwnership")

				// Take ownership of the object
				result := TakeOwnership(tc.input, tc.manager)

				assert.Equal(t, tc.expectedLabels, result.GetLabels(),
					"Object should now have expected labels after TakeOwnership")

				// Check if the object should be managed after taking ownership
				afterManage := ShouldManage(result, tc.manager)
				assert.Equal(t, !tc.expectedShouldManage, afterManage)
			}
		})
	}
}
