package consts

const (
	SecretKeyMetricsData         = "payload"
	SecretKeyRegistrationCode    = "regCode"
	SecretKeyOfflineRegRequest   = "request"
	SecretKeyOfflineRegCert      = "certificate"
	SecretKeyRegistrationURL     = "registrationUrl"
	SecretKeyRegistrationURLCert = "registrationUrlCert"
	SecretKeyInstanceData        = "instanceData"
)

type SecretRole string

const (
	SCCCredentialsRole           SecretRole = "scc-credentials"
	RegistrationCode             SecretRole = "reg-code"
	OfflineRequestRole           SecretRole = "offline-request"
	OfflineCertificate           SecretRole = "offline-certificate"
	RegistrationServerCertRole   SecretRole = "registration-server-cert"
	RegistrationInstanceDataRole SecretRole = "registration-instance-data"
)
