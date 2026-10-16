package suseconnect

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/rancher/scc-operator/internal/consts"
	"github.com/rancher/scc-operator/internal/repos/secretrepo"
	"github.com/rancher/wrangler/v3/pkg/generic/fake"
)

func TestFetchInstanceDataFrom(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockController := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
	mockCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)

	tests := []struct {
		name      string
		secretRef *corev1.SecretReference
		setupMock func()
		expected  []byte
	}{
		{
			name:      "returns nil when secretRef is nil",
			secretRef: nil,
			setupMock: func() {},
			expected:  nil,
		},
		{
			name: "returns instance data when secret exists with data",
			secretRef: &corev1.SecretReference{
				Name:      "instance-data-secret",
				Namespace: "test-namespace",
			},
			setupMock: func() {
				secret := &corev1.Secret{
					Data: map[string][]byte{
						consts.SecretKeyInstanceData: []byte(`<document><instance_data>test</instance_data></document>`),
					},
				}
				mockCache.EXPECT().
					Get("test-namespace", "instance-data-secret").
					Return(secret, nil).
					Times(1)
			},
			expected: []byte(`<document><instance_data>test</instance_data></document>`),
		},
		{
			name: "returns nil when secret not found",
			secretRef: &corev1.SecretReference{
				Name:      "missing-secret",
				Namespace: "test-namespace",
			},
			setupMock: func() {
				mockCache.EXPECT().
					Get("test-namespace", "missing-secret").
					Return(nil, apierrors.NewNotFound(schema.GroupResource{}, "missing-secret")).
					Times(1)
			},
			expected: nil,
		},
		{
			name: "returns nil when secret exists but has no instance data key",
			secretRef: &corev1.SecretReference{
				Name:      "incomplete-secret",
				Namespace: "test-namespace",
			},
			setupMock: func() {
				secret := &corev1.Secret{
					Data: map[string][]byte{
						"other-key": []byte("other-data"),
					},
				}
				mockCache.EXPECT().
					Get("test-namespace", "incomplete-secret").
					Return(secret, nil).
					Times(1)
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			repo := &secretrepo.SecretRepository{
				Controller: mockController,
				Cache:      mockCache,
			}

			result := FetchInstanceDataFrom(repo, tt.secretRef)
			assert.Equal(t, tt.expected, result)
		})
	}
}
