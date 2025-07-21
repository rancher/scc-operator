package suseconnect

import (
	"github.com/rancher/scc-operator/internal/consts"
	"github.com/rancher/scc-operator/internal/repos/secretrepo"
	corev1 "k8s.io/api/core/v1"
)

func FetchSccRegistrationCodeFrom(secretRepo *secretrepo.SecretRepository, reference *corev1.SecretReference) string {
	sccContextLogger().Debugf("Fetching SCC Registration Code from secret %s/%s", reference.Namespace, reference.Name)
	regSecret, err := secretRepo.Cache.Get(reference.Namespace, reference.Name)
	if err != nil {
		sccContextLogger().Warnf("Failed to get SCC Registration Code from secret %s/%s: %v", reference.Namespace, reference.Name, err)
		return ""
	}
	sccContextLogger().Debugf("Found secret %s/%s", reference.Namespace, reference.Name)

	regCode, ok := regSecret.Data[consts.SecretKeyRegistrationCode]
	if !ok {
		sccContextLogger().Warnf("registration secret `%v` does not contain expected data `%s`", reference, consts.SecretKeyRegistrationCode)
		return ""
	}

	return string(regCode)
}
