package config

import (
	"context"
	"fmt"
	"strconv"
	"sync"

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
	OperatorName    string
	Kubeconfig      string
	SystemNamespace string
	LeaseNamespace  string
	LogFormat       rootLog.Format
	LogLevel        logrus.Level
	CattleDevMode   bool

	// DevMode tracks the operators "dev mode" status, when enabled many features will be configured for better dev feedback
	DevMode               bool
	DefaultSCCEnvironment consts.SCCEnvironment
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

// Global variable for live configuration.
var (
	currentConfig *OperatorSettings
	mu            sync.RWMutex
)

// LoadInitialConfig will fetch a value resolver and combine it with a ConfigMap (if exists) to prepare an OperatorSettings
func LoadInitialConfig(ctx context.Context) (*OperatorSettings, error) {
	valueResolver := NewValueResolver()

	kubeconfigPath := valueResolver.Get(Kubeconfig, "")

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
	devMode, _ := strconv.ParseBool(valueResolver.Get(DevMode, "false"))

	loadedConfig := &OperatorSettings{
		Kubeconfig:      kubeconfigPath,
		OperatorName:    valueResolver.Get(OperatorName, consts.DefaultOperatorName),
		SystemNamespace: operatorNamespace,
		LeaseNamespace:  valueResolver.Get(LeaseNamespace, consts.DefaultLeaseNamespace),
		LogFormat:       decideLogFormat(valueResolver.Get(LogFormat, "")),
		LogLevel:        decideLogLevel(loggingLevel, trace, debug),
		CattleDevMode:   valueResolver.Get(RancherDevMode, "") != "",
		DevMode:         devMode,
	}

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
