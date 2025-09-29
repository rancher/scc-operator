package helpers

import (
	"fmt"
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

// HelpersTestCase defines a unified test case for all helper functions
type HelpersTestCase struct {
	name                       string
	input                      runtimeMetaObject
	owner                      string // Used for TakeOwnership and ShouldManage
	expectedHasManaged         bool
	expectedShouldManage       bool
	expectedAfterTakeOwnership bool              // For TakeOwnership result
	expectedLabels             map[string]string // For TakeOwnership result
}

// getTestCases returns a list of test cases that can be used for all helper functions
func getTestCases() []HelpersTestCase {
	return []HelpersTestCase{
		// Secret test cases
		{
			name: "secret with managed-by annotation",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						consts.LabelSccManagedBy: "test-value",
					},
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "test-manager",
					},
				},
			},
			owner:                      "test-manager",
			expectedHasManaged:         true,
			expectedShouldManage:       true,
			expectedAfterTakeOwnership: true,
			expectedLabels: map[string]string{
				consts.LabelK8sManagedBy: "test-manager",
			},
		},
		{
			name: "secret with other annotations",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"other-annotation": "some-value",
					},
					Labels: map[string]string{
						"other-label": "some-value",
					},
				},
			},
			owner:                      "test-manager",
			expectedHasManaged:         false,
			expectedShouldManage:       false,
			expectedAfterTakeOwnership: true,
			expectedLabels: map[string]string{
				"other-label":            "some-value",
				consts.LabelK8sManagedBy: "test-manager",
			},
		},
		{
			name: "secret with no annotations",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-name",
				},
			},
			owner:                      "test-manager",
			expectedHasManaged:         false,
			expectedShouldManage:       false,
			expectedAfterTakeOwnership: true,
			expectedLabels: map[string]string{
				consts.LabelK8sManagedBy: "test-manager",
			},
		},
		// ConfigMap test cases
		{
			name: "configmap with managed-by annotation",
			input: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						consts.LabelSccManagedBy: "test-value",
					},
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "test-manager",
					},
				},
			},
			owner:                      "test-manager",
			expectedHasManaged:         true,
			expectedShouldManage:       true,
			expectedAfterTakeOwnership: true,
			expectedLabels: map[string]string{
				consts.LabelK8sManagedBy: "test-manager",
			},
		},
		{
			name: "configmap with other annotations",
			input: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"other-annotation": "some-value",
					},
					Labels: map[string]string{
						"other-label": "some-value",
					},
				},
			},
			owner:                      "test-manager",
			expectedHasManaged:         false,
			expectedShouldManage:       false,
			expectedAfterTakeOwnership: true,
			expectedLabels: map[string]string{
				"other-label":            "some-value",
				consts.LabelK8sManagedBy: "test-manager",
			},
		},
		{
			name: "configmap with no annotations",
			input: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-name",
				},
			},
			owner:                      "test-manager",
			expectedHasManaged:         false,
			expectedShouldManage:       false,
			expectedAfterTakeOwnership: true,
			expectedLabels: map[string]string{
				consts.LabelK8sManagedBy: "test-manager",
			},
		},
		// Registration CRD test cases
		{
			name: "registration with managed-by annotation",
			input: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						consts.LabelSccManagedBy: "test-value",
					},
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "test-manager",
					},
				},
			},
			owner:                      "test-manager",
			expectedHasManaged:         true,
			expectedShouldManage:       true,
			expectedAfterTakeOwnership: true,
			expectedLabels: map[string]string{
				consts.LabelK8sManagedBy: "test-manager",
			},
		},
		{
			name: "registration with other annotations",
			input: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"other-annotation": "some-value",
					},
					Labels: map[string]string{
						"other-label": "some-value",
					},
				},
			},
			owner:                      "test-manager",
			expectedHasManaged:         false,
			expectedShouldManage:       false,
			expectedAfterTakeOwnership: true,
			expectedLabels: map[string]string{
				"other-label":            "some-value",
				consts.LabelK8sManagedBy: "test-manager",
			},
		},
		{
			name: "registration with no annotations",
			input: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-reg",
				},
			},
			owner:                      "test-manager",
			expectedHasManaged:         false,
			expectedShouldManage:       false,
			expectedAfterTakeOwnership: true,
			expectedLabels: map[string]string{
				consts.LabelK8sManagedBy: "test-manager",
			},
		},
		// Edge cases
		{
			name: "object with nil annotations/labels map",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   "test-namespace",
					Name:        "test-name",
					Annotations: nil,
					Labels:      nil,
				},
			},
			owner:                      "test-manager",
			expectedHasManaged:         false,
			expectedShouldManage:       false,
			expectedAfterTakeOwnership: true,
			expectedLabels: map[string]string{
				consts.LabelK8sManagedBy: "test-manager",
			},
		},
		{
			name: "object with different manager",
			input: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelK8sManagedBy: "other-manager",
					},
				},
			},
			owner:                      "test-manager",
			expectedHasManaged:         false,
			expectedShouldManage:       false,
			expectedAfterTakeOwnership: true,
			expectedLabels: map[string]string{
				consts.LabelK8sManagedBy: "test-manager",
			},
		},
		{
			name:                       "empty secret object",
			input:                      &corev1.Secret{},
			owner:                      "",
			expectedHasManaged:         false,
			expectedShouldManage:       false,
			expectedAfterTakeOwnership: false,
			expectedLabels:             nil,
		},
		{
			name:                       "empty registration object",
			input:                      &v1.Registration{},
			owner:                      "",
			expectedHasManaged:         false,
			expectedShouldManage:       false,
			expectedAfterTakeOwnership: false,
			expectedLabels:             nil,
		},
	}
}

func TestHasManagedByLabel(t *testing.T) {
	testCases := getTestCases()

	for _, tc := range testCases {
		tc := tc // capture range variable for parallel execution
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := HasManagedByLabel(tc.input)
			assert.Equal(t, tc.expectedHasManaged, result)
		})
	}
}

func TestShouldManage(t *testing.T) {
	testCases := getTestCases()

	for _, tc := range testCases {
		tc := tc // capture range variable for parallel execution
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := ShouldManage(tc.input, tc.owner)
			assert.Equal(t, tc.expectedShouldManage, result)
		})
	}
}

func TestTakeOwnership(t *testing.T) {
	testCases := getTestCases()

	for _, tc := range testCases {
		tc := tc // capture range variable for parallel execution
		t.Run(tc.name, func(t *testing.T) {
			//t.Parallel()

			// Check if the object should be managed before taking ownership
			beforeManage := ShouldManage(tc.input, tc.owner)
			if beforeManage {
				// If the object already had the correct owner, it should have been managed before
				assert.Equal(t, tc.owner, tc.input.GetLabels()[consts.LabelK8sManagedBy],
					"Object should have had the correct owner before taking ownership")
			} else {
				// Only when take ownership is a success should we assert what labels exist before
				if tc.expectedAfterTakeOwnership {
					assert.NotEqual(t, tc.expectedLabels, tc.input.GetLabels(),
						"Object should not have expected labels until after TakeOwnership")
				}

				// Take ownership of the object
				result := TakeOwnership(tc.input, tc.owner)

				assert.Equal(t, tc.expectedLabels, tc.input.GetLabels(),
					"Object should not have expected labels until after TakeOwnership")

				// Check if the object should be managed after taking ownership
				afterManage := ShouldManage(result, tc.owner)

				expectedState := "managed"
				if !tc.expectedAfterTakeOwnership {
					expectedState = "unmanaged"
				}
				// After taking ownership, the object should always be managed
				assert.Equal(t, afterManage, tc.expectedAfterTakeOwnership, fmt.Sprintf("Object should be %s after taking ownership", expectedState))
			}
		})
	}
}
