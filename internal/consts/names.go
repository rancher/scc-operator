package consts

import "fmt"

// Secret names and name prefixes
const (
	ResourceSCCEntrypointSecretName      = "scc-registration"
	SCCMetricsOutputSecretName           = "rancher-scc-metrics"
	RancherMetricsSecretRequestName      = SCCMetricsOutputSecretName
	SCCSystemCredentialsSecretNamePrefix = "scc-system-credentials-"
	RegistrationCodeSecretNamePrefix     = "registration-code-"
	OfflineRequestSecretNamePrefix       = "offline-request-"
	OfflineCertificateSecretNamePrefix   = "offline-certificate-"
)

func RegistrationName(namePartIn string) string {
	return fmt.Sprintf("%s-%s", ResourceSCCEntrypointSecretName, namePartIn)
}

func SCCCredentialsSecretName(namePartIn string) string {
	return fmt.Sprintf("%s%s", SCCSystemCredentialsSecretNamePrefix, namePartIn)
}

func RegistrationCodeSecretName(namePartIn string) string {
	return fmt.Sprintf("%s%s", RegistrationCodeSecretNamePrefix, namePartIn)
}

func OfflineRequestSecretName(namePartIn string) string {
	return fmt.Sprintf("%s%s", OfflineRequestSecretNamePrefix, namePartIn)
}

func OfflineCertificateSecretName(namePartIn string) string {
	return fmt.Sprintf("%s%s", OfflineCertificateSecretNamePrefix, namePartIn)
}
