package offline

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/rancher/scc-operator/internal/repos/secretrepo"
	"github.com/rancher/scc-operator/pkg/controllers/shared"
)

type SecretManager struct {
	secretNamespace       string
	requestSecretName     string
	certificateSecretName string
	ownerRef              *metav1.OwnerReference
	secretRepo            *secretrepo.SecretRepository
	offlineRequest        []byte
	defaultLabels         map[string]string
}

func New(
	namespace, requestName, certificateName string,
	ownerRef *metav1.OwnerReference,
	secretRepo *secretrepo.SecretRepository,
	labels map[string]string,
) *SecretManager {
	return &SecretManager{
		secretNamespace:       namespace,
		requestSecretName:     requestName,
		certificateSecretName: certificateName,
		secretRepo:            secretRepo,
		ownerRef:              ownerRef,
		defaultLabels:         labels,
	}
}

func (o *SecretManager) Remove() error {
	certErr := o.RemoveOfflineCertificate()
	requestErr := o.RemoveOfflineRequest()

	if requestErr != nil && certErr != nil {
		return fmt.Errorf("failed to remove both offline request & certificate: %v; %v", requestErr, certErr)
	}
	if certErr != nil {
		return fmt.Errorf("failed to remove offline certificate: %v", certErr)
	}
	if requestErr != nil {
		return fmt.Errorf("failed to remove offline request: %v", requestErr)
	}

	return nil
}

func (o *SecretManager) removeOfflineFinalizer(incomingSecret *corev1.Secret) error {
	if shared.SecretHasOfflineFinalizer(incomingSecret) {
		updatedSecret := incomingSecret.DeepCopy()
		updatedSecret = shared.SecretRemoveOfflineFinalizer(updatedSecret)
		_, updateErr := o.secretRepo.CreateOrUpdateSecret(updatedSecret)

		if updateErr == nil || apierrors.IsNotFound(updateErr) {
			return nil
		}
		if !apierrors.IsConflict(updateErr) {
			return updateErr
		}
		return retry.RetryOnConflict(retry.DefaultRetry, func() error {
			currentSecret, getErr := o.secretRepo.Get(incomingSecret.Namespace, incomingSecret.Name)
			if getErr != nil && !apierrors.IsNotFound(getErr) {
				return getErr
			}
			updatedSecret := currentSecret.DeepCopy()
			updatedSecret = shared.SecretRemoveOfflineFinalizer(updatedSecret)
			var updateErr error
			_, updateErr = o.secretRepo.PatchUpdate(currentSecret, updatedSecret)
			return updateErr
		})
	}

	return nil
}
