package lifecycle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/scc-operator/internal/consts"
)

func TestSecretHasInstanceDataFinalizer(t *testing.T) {
	tests := []struct {
		name     string
		secret   *corev1.Secret
		expected bool
	}{
		{
			name: "secret has instance data finalizer",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{consts.FinalizerRMTInstanceData},
				},
			},
			expected: true,
		},
		{
			name: "secret has instance data finalizer among others",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{
						"other-finalizer",
						consts.FinalizerRMTInstanceData,
						"another-finalizer",
					},
				},
			},
			expected: true,
		},
		{
			name: "secret does not have instance data finalizer",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{consts.FinalizerSccRegistrationCode},
				},
			},
			expected: false,
		},
		{
			name: "secret has no finalizers",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SecretHasInstanceDataFinalizer(tt.secret)
			assert.Equal(t, tt.expected, result)
		})
	}
}
