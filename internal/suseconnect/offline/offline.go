package offline

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/scc-operator/internal/repos/secretrepo"
	"github.com/rancher/scc-operator/pkg/controllers/common"
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
	if common.SecretHasOfflineFinalizer(incomingSecret) {
		updatedSecret := incomingSecret.DeepCopy()
		updatedSecret = common.SecretRemoveOfflineFinalizer(updatedSecret)
		if _, updateErr := o.secretRepo.CreateOrUpdateSecret(updatedSecret); updateErr != nil {
			if apierrors.IsNotFound(updateErr) {
				return nil
			}

			return updateErr
		}
	}

	return nil
}
