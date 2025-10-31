package consts

const (
	DefaultOperatorName      = "rancher-scc-operator" // TODO: in the future when this isn't very specific to `rancher` (the product) drop the `rancher-` prefix
	DefaultSCCNamespace      = "cattle-scc-system"
	DefaultLeaseNamespace    = "kube-system"
	SCCOperatorConfigMapName = "scc-operator-config"
)

const (
	FinalizerSccMetricsSecretRequest = "scc.cattle.io/scc-metrics-request"
	FinalizerSccOfflineSecret        = "scc.cattle.io/managed-offline-secret"
	FinalizerSccCredentials          = "scc.cattle.io/managed-credentials"
	FinalizerSccRegistration         = "scc.cattle.io/managed-registration"
	FinalizerSccRegistrationCode     = "scc.cattle.io/managed-registration-code"
)

const (
	// LabelK8sManagedBy identifies "the tool being used to manage the operation of an application" (per k8s docs).
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

// These are consts for Rancher setting names we may need to lookup
const (
	SettingNameInstallUUID   = "install-uuid"
	SettingNameServerURL     = "server-url"
	SettingNameServerVersion = "server-version"
)

const OperatorWorkerThreads = 2
