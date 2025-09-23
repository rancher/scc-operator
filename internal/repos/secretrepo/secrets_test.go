package secretrepo

import (
	"errors"
	"testing"

	"github.com/rancher/wrangler/v3/pkg/generic/fake"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/rancher/scc-operator/internal/consts"
)

// test helper to create a basic secret
func newSecret(ns, name string, data map[string][]byte) *corev1.Secret {
	if ns == "" {
		ns = "default"
	}
	if name == "" {
		name = "test-secret"
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Data:       data,
	}
}

func TestNewSecretRepository(t *testing.T) {
	asserts := assert.New(t)
	// Ensure we start with a clean singleton for this test
	rootSecretRepo = nil

	ctrl := gomock.NewController(t)
	mockController := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
	mockCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)
	// Expect indexers to be added during initialization
	mockCache.EXPECT().AddIndexer(IndexSecretsByPath, gomock.Any()).Times(1)
	mockCache.EXPECT().AddIndexer(IndexSecretsBySccHash, gomock.Any()).Times(1)
	repo1 := NewSecretRepository("testing-namespace", mockController, mockCache)
	// Should initialize singleton and set fields
	asserts.NotNil(repo1)
	asserts.Equal(mockController, repo1.Controller)
	asserts.Equal(mockCache, repo1.Cache)

	// Subsequent calls should return the same singleton instance and not override fields
	ctrl2 := gomock.NewController(t)
	otherController := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl2)
	otherCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl2)
	repo2 := NewSecretRepository("testing-namespace", otherController, otherCache)
	asserts.Same(repo1, repo2)
	asserts.Equal(mockController, repo2.Controller)
	asserts.Equal(mockCache, repo2.Cache)
}

func TestHasSecret(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockController := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
	mockCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)

	repo := &SecretRepository{Controller: mockController, Cache: mockCache}

	ns, name := "n", "s"
	// Cache returns secret -> HasSecret true
	mockCache.EXPECT().Get(ns, name).Return(newSecret(ns, name, nil), nil).Times(1)
	assert.True(t, repo.HasSecret(ns, name))

	// Cache returns NotFound -> HasSecret false
	notFound := apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "secrets"}, name)
	mockCache.EXPECT().Get(ns, name).Return(nil, notFound).Times(1)
	assert.False(t, repo.HasSecret(ns, name))
}

func TestGet_UsesCacheThenControllerOnNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockController := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
	mockCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)

	repo := &SecretRepository{Controller: mockController, Cache: mockCache}

	ns, name := "n", "s"

	// Case 1: Cache hit returns immediately
	fromCache := newSecret(ns, name, nil)
	mockCache.EXPECT().Get(ns, name).Return(fromCache, nil).Times(1)
	got, err := repo.Get(ns, name)
	assert.NoError(t, err)
	assert.Equal(t, fromCache, got)

	// Case 2: Cache NotFound -> controller Get
	notFound := apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "secrets"}, name)
	fromAPI := newSecret(ns, name, map[string][]byte{"a": []byte("b")})
	mockCache.EXPECT().Get(ns, name).Return(nil, notFound).Times(1)
	mockController.EXPECT().Get(ns, name, metav1.GetOptions{}).Return(fromAPI, nil).Times(1)
	got, err = repo.Get(ns, name)
	assert.NoError(t, err)
	assert.Equal(t, fromAPI, got)
}

func TestPatchUpdate_CallsControllerPatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockController := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
	mockCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)

	repo := &SecretRepository{Controller: mockController, Cache: mockCache}

	incoming := newSecret("n", "s", map[string][]byte{"k": []byte("v1")})
	desired := newSecret("n", "s", map[string][]byte{"k": []byte("v2")})

	// We don't validate patch contents; ensure merge patch type and proper ns/name
	updated := desired.DeepCopy()
	mockController.EXPECT().Patch(incoming.Namespace, incoming.Name, types.MergePatchType, gomock.Any()).Return(updated, nil).Times(1)

	got, err := repo.PatchUpdate(incoming, desired)
	assert.NoError(t, err)
	assert.Equal(t, updated, got)
}

func TestPatchUpdate_JSONMarshalErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockController := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
	mockCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)

	repo := &SecretRepository{Controller: mockController, Cache: mockCache}

	incoming := newSecret("n", "s", map[string][]byte{"k": []byte("v1")})
	desired := newSecret("n", "s", map[string][]byte{"k": []byte("v2")})

	// Preserve original jsonMarshal and restore after test
	origMarshal := jsonMarshal
	defer func() { jsonMarshal = origMarshal }()

	// We use a function builder to reuse code that targets conditional json failures
	// Because json.Marshal is called twice within `PatchUpdate`
	failingJsonMarshalBuilder := func(targetCallNumber int) func(v any) ([]byte, error) {
		calls := 0
		return func(v any) ([]byte, error) {
			calls++
			if calls == targetCallNumber {
				errorText := "marshal incoming failed"
				if targetCallNumber == 2 {
					errorText = "marshal desired failed"
				}
				return nil, errors.New(errorText)
			}
			return origMarshal(v)
		}
	}

	t.Run("error marshalling incoming", func(t *testing.T) {
		jsonMarshal = failingJsonMarshalBuilder(1) // Target incoming data's json.Marshal

		updated, err := repo.PatchUpdate(incoming, desired)
		assert.Error(t, err)
		assert.Equal(t, "marshal incoming failed", err.Error())
		assert.Same(t, incoming, updated)
	})

	t.Run("error marshalling desired", func(t *testing.T) {
		jsonMarshal = failingJsonMarshalBuilder(2) // Target desired data's json.Marshal

		updated, err := repo.PatchUpdate(incoming, desired)
		assert.Error(t, err)
		assert.Equal(t, "marshal desired failed", err.Error())
		assert.Same(t, incoming, updated)
	})
}

func TestRetryingPatchUpdate_FirstFailsThenRetrySucceeds(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockController := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
	mockCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)

	repo := &SecretRepository{Controller: mockController, Cache: mockCache}

	incoming := newSecret("n", "s", map[string][]byte{"k": []byte("v1")})
	desired := newSecret("n", "s", map[string][]byte{"k": []byte("v2")})

	// First Patch (from initial PatchUpdate) fails
	mockController.EXPECT().Patch(incoming.Namespace, incoming.Name, types.MergePatchType, gomock.Any()).Return(nil, errors.New("boom")).Times(1)

	// Retry: Get current secret succeeds
	current := newSecret("n", "s", map[string][]byte{"k": []byte("v1-current")})
	mockController.EXPECT().Get(incoming.Namespace, incoming.Name, metav1.GetOptions{}).Return(current, nil).Times(1)

	// Retry Patch succeeds
	updated := desired.DeepCopy()
	mockController.EXPECT().Patch(current.Namespace, current.Name, types.MergePatchType, gomock.Any()).Return(updated, nil).Times(1)

	got, err := repo.RetryingPatchUpdate(incoming, desired)
	assert.NoError(t, err)
	assert.Equal(t, updated, got)
}

func TestCreateOrUpdateSecret_CreateOnNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockController := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
	mockCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)

	repo := &SecretRepository{Controller: mockController, Cache: mockCache}

	ns, name := "n", "s"
	desired := newSecret(ns, name, map[string][]byte{"k": []byte("v")})

	notFound := apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "secrets"}, name)
	mockCache.EXPECT().Get(ns, name).Return(nil, notFound).Times(1)
	mockController.EXPECT().Create(desired).Return(desired, nil).Times(1)

	got, err := repo.CreateOrUpdateSecret(desired)
	assert.NoError(t, err)
	assert.Equal(t, desired, got)
}

func TestCreateOrUpdateSecret_UpdateWhenExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockController := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
	mockCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)

	repo := &SecretRepository{Controller: mockController, Cache: mockCache}

	ns, name := "n", "s"
	existing := newSecret(ns, name, map[string][]byte{"k": []byte("v1")})
	desired := newSecret(ns, name, map[string][]byte{"k": []byte("v2")})

	// Cache has existing
	mockCache.EXPECT().Get(ns, name).Return(existing, nil).Times(1)
	// RetryingPatchUpdate path: initial patch error then success
	mockController.EXPECT().Patch(ns, name, types.MergePatchType, gomock.Any()).Return(nil, errors.New("conflict")).Times(1)
	mockController.EXPECT().Get(ns, name, metav1.GetOptions{}).Return(existing, nil).Times(1)
	updated := desired.DeepCopy()
	mockController.EXPECT().Patch(ns, name, types.MergePatchType, gomock.Any()).Return(updated, nil).Times(1)

	got, err := repo.CreateOrUpdateSecret(desired)
	assert.NoError(t, err)
	assert.Equal(t, updated, got)
}

func TestHasMetricsSecret_And_FetchMetricsSecret(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockController := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
	mockCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)

	repo := &SecretRepository{Controller: mockController, Cache: mockCache}
	ns := "debug-namespace"
	systemIndexNamespace = ns
	name := consts.SCCMetricsOutputSecretName

	payload := []byte(`{"hello":"world","count":5,"subscription":{"installuuid":"5","product":"ranchdressing","version":"42","arch":"unknown","git":"no thanks"}}`)
	sec := newSecret(ns, name, map[string][]byte{consts.SecretKeyMetricsData: payload})

	// HasMetricsSecret uses Cache.Get
	mockCache.EXPECT().Get(ns, name).Return(sec, nil).Times(1)
	assert.True(t, repo.HasMetricsSecret())

	// FetchMetricsSecret uses Get: prefer Cache hit
	mockCache.EXPECT().Get(ns, name).Return(sec, nil).Times(1)
	w, err := repo.FetchMetricsSecret()
	assert.NoError(t, err)
	// Not asserting internal state of wrapper, only that no error occurred and wrapper is non-zero
	var zeroWrapper interface{} = w
	assert.NotNil(t, zeroWrapper)

	// Error: missing metrics key
	secNoKey := newSecret(ns, name, map[string][]byte{"other": []byte("x")})
	mockCache.EXPECT().Get(ns, name).Return(secNoKey, nil).Times(1)
	_, err = repo.FetchMetricsSecret()
	assert.Error(t, err)

	// Error: invalid JSON
	secBadJSON := newSecret(ns, name, map[string][]byte{consts.SecretKeyMetricsData: []byte("not-json")})
	mockCache.EXPECT().Get(ns, name).Return(secBadJSON, nil).Times(1)
	_, err = repo.FetchMetricsSecret()
	assert.Error(t, err)
}
