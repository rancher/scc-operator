package consts

import (
	"fmt"

	"github.com/rancher/scc-operator/internal/initializer"
)

const (
	// DefaultOperatorName TODO: in the future when this isn't very specific to `rancher` (the product) drop the `rancher-` prefix
	DefaultOperatorName = "rancher-scc-operator"
	DefaultSCCNamespace = "cattle-scc-system"
)

// Secret names and name prefixes
const (
	ResourceSCCEntrypointSecretName      = "scc-registration"
	SCCMetricsOutputSecretName           = "rancher-scc-metrics"
	SCCSystemCredentialsSecretNamePrefix = "scc-system-credentials-"
	RegistrationCodeSecretNamePrefix     = "registration-code-"
	OfflineRequestSecretNamePrefix       = "offline-request-"
	OfflineCertificateSecretNamePrefix   = "offline-certificate-"
)

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

const (
	ManagedBySecretBroker = "secret-broker"
)

const (
	FinalizerSccOfflineSecret    = "scc.cattle.io/managed-offline-secret"
	FinalizerSccCredentials      = "scc.cattle.io/managed-credentials"
	FinalizerSccRegistration     = "scc.cattle.io/managed-registration"
	FinalizerSccRegistrationCode = "scc.cattle.io/managed-registration-code"
)

const (
	LabelK8sManagedBy = "app.kubernetes.io/managed-by"

	LabelObjectSalt       = "scc.cattle.io/instance-salt"
	LabelNameSuffix       = "scc.cattle.io/related-name-suffix"
	LabelSccHash          = "scc.cattle.io/scc-hash"
	LabelSccLastProcessed = "scc.cattle.io/last-processed"

	// LabelSccManagedBy identifies the name of the SCC operator that manages a specific resource
	LabelSccManagedBy  = "scc.cattle.io/managed-by"
	LabelSccSecretRole = "scc.cattle.io/secret-role"
)

const (
	SecretKeyRegistrationCode  = "regCode"
	SecretKeyOfflineRegRequest = "request"
	SecretKeyOfflineRegCert    = "certificate"
	RegistrationURL            = "registrationUrl"
)

type SecretRole string

const (
	SCCCredentialsRole SecretRole = "scc-credentials"
	RegistrationCode   SecretRole = "reg-code"
	OfflineRequestRole SecretRole = "offline-request"
	OfflineCertificate SecretRole = "offline-certificate"
)

type SCCEnvironment int

const (
	Production SCCEnvironment = iota
	Staging
	PayAsYouGo
	RGS
)

func (s SCCEnvironment) String() string {
	switch s {
	case Production:
		return "production"
	case Staging:
		return "staging"
	case PayAsYouGo:
		return "payAsYouGo"
	case RGS:
		return "rgs"
	default:
		return "unknown"
	}
}

func GetSCCEnvironment() SCCEnvironment {
	if !initializer.DevMode.Get() {
		return Production
	}
	return Staging
}

type AlternativeSccURLs string

const (
	ProdSccURL    AlternativeSccURLs = "https://scc.suse.com"
	StagingSccURL AlternativeSccURLs = "https://stgscc.suse.com"
)

// TODO in the future we can store the PAYG and other urls too

func (s AlternativeSccURLs) Ptr() *string {
	stringVal := string(s)
	return &stringVal
}

func BaseURLForSCC() string {
	var baseURL string
	switch GetSCCEnvironment() {
	case Production:
		baseURL = string(ProdSccURL)
	case Staging:
		baseURL = string(StagingSccURL)
	case RGS: // explicitly return empty for RGS
	default:
		// intentionally do nothing and return empty string
	}

	return baseURL
}
