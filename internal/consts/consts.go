package consts

const (
	// DefaultOperatorName TODO: in the future when this isn't very specific to `rancher` (the product) drop the `rancher-` prefix
	DefaultOperatorName = "rancher-scc-operator"
	DefaultSCCNamespace = "cattle-scc-system"
)

const (
	FinalizerSccMetricsSecretRequest = "scc.cattle.io/scc-metrics-request"
	FinalizerSccOfflineSecret        = "scc.cattle.io/managed-offline-secret"
	FinalizerSccCredentials          = "scc.cattle.io/managed-credentials"
	FinalizerSccRegistration         = "scc.cattle.io/managed-registration"
	FinalizerSccRegistrationCode     = "scc.cattle.io/managed-registration-code"
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
	ManagedByValueSecretBroker = "secret-broker"
)

const (
	SettingNameInstallUUID   = "install-uuid"
	SettingNameServerURL     = "server-url"
	SettingNameServerVersion = "server-version"
)
