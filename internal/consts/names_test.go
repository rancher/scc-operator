package consts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSCCCredentialsSecretName(t *testing.T) {
	asserts := assert.New(t)

	asserts.Equal("scc-system-credentials-", SCCCredentialsSecretName(""))
	asserts.Equal("scc-system-credentials-test", SCCCredentialsSecretName("test"))
}

func TestRegistrationCodeSecretName(t *testing.T) {
	asserts := assert.New(t)

	asserts.Equal("registration-code-", RegistrationCodeSecretName(""))
	asserts.Equal("registration-code-test", RegistrationCodeSecretName("test"))
}

func TestOfflineRequestSecretName(t *testing.T) {
	asserts := assert.New(t)

	asserts.Equal("offline-request-", OfflineRequestSecretName(""))
	asserts.Equal("offline-request-test", OfflineRequestSecretName("test"))
}

func TestOfflineCertificateSecretName(t *testing.T) {
	asserts := assert.New(t)

	asserts.Equal("offline-certificate-", OfflineCertificateSecretName(""))
	asserts.Equal("offline-certificate-test", OfflineCertificateSecretName("test"))
}
