package secretrepo

import (
	jsonpatch "github.com/evanphx/json-patch/v5"
	corev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/util/retry"

	"github.com/rancher/scc-operator/internal/repos/generic"
)

var rootSecretRepo *SecretRepository

type SecretRepository generic.RuntimeObjectRepo[*v1.Secret, *v1.SecretList]

func NewSecretRepository(
	secrets corev1.SecretController,
	secretsCache corev1.SecretCache,
) *SecretRepository {
	if rootSecretRepo == nil {
		rootSecretRepo = &SecretRepository{
			Controller: secrets,
			Cache:      secretsCache,
		}
		rootSecretRepo.InitIndexers()
	}

	return rootSecretRepo
}

func (r *SecretRepository) HasSecret(namespace, name string) bool {
	_, err := r.Cache.Get(namespace, name)
	return err == nil
}

func (r *SecretRepository) PatchUpdate(incoming, desired *v1.Secret) (*v1.Secret, error) {
	incomingJson, err := json.Marshal(incoming)
	if err != nil {
		return incoming, err
	}
	newJson, err := json.Marshal(desired)
	if err != nil {
		return incoming, err
	}

	patch, err := jsonpatch.CreateMergePatch(incomingJson, newJson)
	if err != nil {
		return incoming, err
	}
	updated, err := r.Controller.Patch(incoming.Namespace, incoming.Name, types.MergePatchType, patch)
	if err != nil {
		return incoming, err
	}

	return updated, nil
}

func (r *SecretRepository) RetryingPatchUpdate(incoming, desired *v1.Secret) (*v1.Secret, error) {
	initialPatched, err := r.PatchUpdate(incoming, desired)
	if err == nil {
		return initialPatched, nil
	}

	var updated *v1.Secret
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		currentSecret, getErr := r.Controller.Get(incoming.Namespace, incoming.Name, metav1.GetOptions{})
		if getErr != nil && !apierrors.IsNotFound(getErr) {
			return getErr
		}

		var updateErr error
		updated, updateErr = r.PatchUpdate(currentSecret, desired)
		return updateErr
	})

	return updated, retryErr
}

func (r *SecretRepository) CreateOrUpdateSecret(secret *v1.Secret) (*v1.Secret, error) {
	existingSecret, getErr := r.Cache.Get(secret.Namespace, secret.Name)
	if getErr != nil && apierrors.IsNotFound(getErr) {
		return r.Controller.Create(secret)
	}

	return r.RetryingPatchUpdate(existingSecret, secret)
}

var _ generic.RuntimeObjectRepository = &SecretRepository{}
