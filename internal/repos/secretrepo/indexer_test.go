package secretrepo

import (
	"testing"

	"github.com/rancher/scc-operator/internal/consts"
	"github.com/rancher/scc-operator/internal/initializer"
	"github.com/rancher/wrangler/v3/pkg/generic/fake"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
)

func TestInternal_secretByPath(t *testing.T) {
	asserts := assert.New(t)
	// Ensure we start with a clean singleton for this test
	rootSecretRepo = nil
	initializer.SystemNamespace.Set("testing-system-ns")

	ctrl := gomock.NewController(t)
	mockController := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
	mockCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)
	// Expect indexers to be added during initialization
	mockCache.EXPECT().AddIndexer(IndexSecretsByPath, gomock.Any()).Times(1)
	mockCache.EXPECT().AddIndexer(IndexSecretsBySccHash, gomock.Any()).Times(1)
	_ = NewSecretRepository(mockController, mockCache)

	testSecret := newSecret("testing-system-ns", "some-secret", nil)

	secretPath, err := secretByPath(testSecret)
	asserts.Equal([]string{"testing-system-ns/some-secret"}, secretPath)
	asserts.Nil(err)
}

func TestInternal_secretByPath_withoutProperInit(t *testing.T) {
	asserts := assert.New(t)

	testSecret := newSecret("not-system-ns", "some-secret", nil)

	secretPath, err := secretByPath(testSecret)
	asserts.Nil(secretPath)
	asserts.Nil(err)
}

func TestInternal_secretToHash(t *testing.T) {
	asserts := assert.New(t)

	testSecret := newSecret("testing-system-ns", "some-secret", nil)
	testSecret.Labels = map[string]string{
		consts.LabelSccHash: "my-test-hash",
	}

	// Test a correctly configured one first
	testHash, err := secretToHash(testSecret)
	asserts.Equal([]string{"my-test-hash"}, testHash)
	asserts.Nil(err)

	testHash, err = secretToHash(nil)
	asserts.Nil(testHash)
	asserts.Nil(err)

	testSecret.Labels = map[string]string{}
	testHash, err = secretToHash(testSecret)
	asserts.Equal([]string{}, testHash)
	asserts.Nil(err)
}
