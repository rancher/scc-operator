package suseconnect

import (
	"crypto/x509"
	"encoding/pem"

	corev1 "k8s.io/api/core/v1"

	"github.com/rancher/scc-operator/internal/consts"
	"github.com/rancher/scc-operator/internal/repos/secretrepo"
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

// FetchRegistrationURLCertFrom fetches and parses the registration URL certificate from a secret
func FetchRegistrationURLCertFrom(secretRepo *secretrepo.SecretRepository, reference *corev1.SecretReference) *x509.Certificate {
	if reference == nil {
		return nil
	}

	sccContextLogger().Debugf("Fetching Registration URL Certificate from secret %s/%s", reference.Namespace, reference.Name)
	certSecret, err := secretRepo.Cache.Get(reference.Namespace, reference.Name)
	if err != nil {
		sccContextLogger().Warnf("Failed to get Registration URL Certificate from secret %s/%s: %v", reference.Namespace, reference.Name, err)
		return nil
	}
	sccContextLogger().Debugf("Found certificate secret %s/%s", reference.Namespace, reference.Name)

	certData, ok := certSecret.Data[consts.RegistrationURLCert]
	if !ok {
		sccContextLogger().Warnf("registration URL cert secret `%v` does not contain expected data `%s`", reference, consts.RegistrationURLCert)
		return nil
	}

	// Parse PEM encoded certificate
	block, _ := pem.Decode(certData)
	if block == nil {
		sccContextLogger().Warnf("failed to decode PEM certificate from secret %s/%s", reference.Namespace, reference.Name)
		return nil
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		sccContextLogger().Warnf("failed to parse x509 certificate from secret %s/%s: %v", reference.Namespace, reference.Name, err)
		return nil
	}

	return cert
}
