package config

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/rancher/scc-operator/internal/suseconnect/products"
	"github.com/rancher/wrangler/v3/pkg/kubeconfig"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/rancher/scc-operator/internal/consts"
	rootLog "github.com/rancher/scc-operator/internal/log"
	"github.com/rancher/scc-operator/pkg/util/log"
)

var logger = log.NewComponentLogger("int/config")

// OperatorSettings represents config values that the SCC Operator relies on to run
// These values are either set by: 1. Reading EnvKey vars, or 2. the ConfigMap used by deployers
// This ensures that SCC Operator execution remains more uniform regardless of execution context.
type OperatorSettings struct {
	OperatorName          string
	Kubeconfig            string
	SystemNamespace       string
	LeaseNamespace        string
	LogFormat             rootLog.Format
	LogLevel              logrus.Level
	CattleDevMode         bool
	DevMode               bool
	DefaultSCCEnvironment consts.SCCEnvironment

	// These are both considered deprecated by default - eventually they must become CRD level not Operator level
	Product        products.ProductName
	ProductVersion string
}

// Validate simply validates the configured settings are potentially valid but not if objects exist
func (s *OperatorSettings) Validate() error {
	if s.OperatorName == "" {
		return fmt.Errorf("operator name must be set")
	}
	if s.SystemNamespace == "" {
		return fmt.Errorf("operator must have a valid SCC system namespace")
	}
	if s.LeaseNamespace == "" {
		logger.Warn("operator lease namespace is empty; will default to `kube-system`")
	}
	return nil
}

func (s *OperatorSettings) initSCCProductConfigs(valueResolver *ValueResolver) {
	// Product override flags/envs should only be used in DevMode so lets force DevMode to be set when observed
	productOverride := valueResolver.Get(ProductOverride, "")
	productVersionOverride := valueResolver.Get(ProductVersionOverride, "")
	if productOverride != "" && productVersionOverride != "" {
		s.Product = products.ProductName(productOverride)
		s.ProductVersion = productVersionOverride
		s.DevMode = true
		// Restricting access to Prod SCC in dev mode makes sense
		s.DefaultSCCEnvironment = consts.StagingSCC

		// Important: These overrides are just override the values used in product triplets for SCC - not the metrics values.
		return
	}

	// TODO: do some actual logic to identify the Product and SCC EnvKey
	// For product, maybe we use `/rancherversion` URL? Need to add new field for product tho.
	// In the future, other product specific "version URL lookup" contracts could be setup.
	productVal := valueResolver.Get(Product, "unknown")
	productVersionVal := valueResolver.Get(ProductVersion, "other")

	s.Product = products.ParseProductName(productVal).ProductName()
	s.ProductVersion = productVersionVal
	// For SCC Environment decide based on version found in `/rancherversion` URL
	// TODO use productVersionVal to pick Staging or Prod
	s.DefaultSCCEnvironment = consts.StagingSCC
}

// Global variable for live configuration.
var (
	currentConfig *OperatorSettings
	mu            sync.RWMutex
)

// LoadInitialConfig will fetch a value resolver and combine it with a ConfigMap (if exists) to prepare an OperatorSettings
func LoadInitialConfig(ctx context.Context) (*OperatorSettings, error) {
	valueResolver := NewValueResolver()

	kubeconfigPath := valueResolver.Get(OperatorNamespace, consts.DefaultSCCNamespace)

	restKubeConfig, err := kubeconfig.GetNonInteractiveClientConfig(kubeconfigPath).ClientConfig()
	if err != nil {
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(restKubeConfig)
	if err != nil {
		return nil, err
	}

	operatorNamespace := valueResolver.Get(OperatorNamespace, consts.DefaultSCCNamespace)

	// Fetch the ConfigMap.
	sccConfigMap, err := clientSet.CoreV1().ConfigMaps(operatorNamespace).Get(ctx, consts.SCCOperatorConfigMapName, metav1.GetOptions{})
	if err != nil {
		logger.Printf("Could not get ConfigMap 'operator-config'. Using flag and env values only. Error: %v", err)
		valueResolver.configMapData = make(map[string]string)
	} else {
		valueResolver.configMapData = sccConfigMap.Data
		valueResolver.hasConfigMap = true
	}
	// Only the Options after this may use config map (if it exits)

	loggingLevel := valueResolver.Get(LogLevel, "")
	trace, _ := strconv.ParseBool(valueResolver.Get(Trace, "false"))
	debug, _ := strconv.ParseBool(valueResolver.Get(Debug, "false"))

	loadedConfig := &OperatorSettings{
		Kubeconfig:      kubeconfigPath,
		OperatorName:    valueResolver.Get(OperatorName, consts.DefaultOperatorName),
		SystemNamespace: operatorNamespace,
		LeaseNamespace:  valueResolver.Get(LeaseNamespace, consts.DefaultLeaseNamespace),
		LogFormat:       decideLogFormat(valueResolver.Get(LogFormat, "")),
		LogLevel:        decideLogLevel(loggingLevel, trace, debug),
		// TODO: this is where we eventually add Dev mode and SCC mode settings too
		CattleDevMode: valueResolver.Get(RancherDevMode, "") != "",
		DevMode:       false,
	}

	// TODO: In the future we may have better mechanics for this
	loadedConfig.initSCCProductConfigs(valueResolver)

	// Set the global config and start the watcher.
	mu.Lock()
	currentConfig = loadedConfig
	mu.Unlock()

	// TODO: maybe eventually just start the watcher here? Can be cancelled via context if needed
	// TODO: Or it ends up as a controller based watcher which will be more familiar to others
	logger.Debug("Initial config loaded - a configmap watcher should be setup to ensure reloads are triggered if changed via configmap")

	return loadedConfig, nil
}

func decideLogFormat(formatStr string) rootLog.Format {
	logFormat := rootLog.Format(formatStr)
	if !logFormat.IsValid() {
		logger.Warnf("Invalid log format '%s' provided. Defaulting to '%s'.", formatStr, rootLog.DefaultFormat)
		return rootLog.DefaultFormat
	}

	return logFormat
}

func decideLogLevel(logLevel string, trace, debug bool) logrus.Level {
	if trace {
		return logrus.TraceLevel
	}

	if debug {
		return logrus.DebugLevel
	}

	if parsedLogLevel, err := logrus.ParseLevel(logLevel); err == nil {
		return parsedLogLevel
	}

	return logrus.InfoLevel
}

func GetCurrentConfig() *OperatorSettings {
	mu.RLock()
	defer mu.RUnlock()
	return currentConfig
}

// TODO: add a watcher implementation here? (or as a wrangler watcher?)
