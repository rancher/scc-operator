package consts

const (
	SecretKeyMetricsData       = "payload"
	SecretKeyRegistrationCode  = "regCode"
	SecretKeyOfflineRegRequest = "request"
	SecretKeyOfflineRegCert    = "certificate"
	RegistrationURL            = "registrationUrl"
	RegistrationURLCert        = "registrationUrlCert"
)

type SecretRole string

const (
	SCCCredentialsRole         SecretRole = "scc-credentials"
	RegistrationCode           SecretRole = "reg-code"
	OfflineRequestRole         SecretRole = "offline-request"
	OfflineCertificate         SecretRole = "offline-certificate"
	RegistrationServerCertRole SecretRole = "registration-server-cert"
)
