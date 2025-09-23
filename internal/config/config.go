package config

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/rancher/wrangler/v3/pkg/kubeconfig"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/rancher/scc-operator/internal/consts"
	rootLog "github.com/rancher/scc-operator/internal/log"
	"github.com/rancher/scc-operator/pkg/util/log"
)

var logger = log.NewComponentLogger("scc-config-handler")

type Option string

const (
	OperatorName      Option = "operator-name"
	OperatorNamespace Option = "operator-namespace"
	LeaseNamespace    Option = "lease-namespace"
	LogLevel          Option = "log-level"
	LogFormat         Option = "log-format"
	KubeconfigPath    Option = "kubeconfig-path"
	Debug             Option = "debug"
	Trace             Option = "trace"
)

func (o Option) Env() EnvVar {
	switch o {
	case OperatorName:
		return SCCOperatorNameEnv
	case OperatorNamespace:
		return SCCSystemNamespaceEnv
	case LeaseNamespace:
		return SCCLeaseNamespaceEnv
	case LogLevel:
		return LogLevelEnv
	case LogFormat:
		return LogFormatEnv
	case KubeconfigPath:
		return KubeconfigEnv
	case Debug:
		return DebugEnv
	case Trace:
		return TraceEnv
	}

	logger.Debugf("unknown env var for option: %s", o)
	return ""
}

func (o Option) ConfigMapValue(configMapData map[string]string) string {
	configMapKey := string(o)
	value := configMapData[configMapKey]
	if value == "" {
		logger.Debugf("unknown config map value for option: %s", o)
	}

	return value
}

type FlagValues struct {
	KubeconfigPath    string
	OperatorName      string
	OperatorNamespace string
	LeaseNamespace    string
	LogLevel          string
	LogFormat         string
	Debug             bool
	Trace             bool
}

func (f *FlagValues) ValueByOption(o Option) string {
	switch o {
	case KubeconfigPath:
		return f.KubeconfigPath
	case OperatorName:
		return f.OperatorName
	case OperatorNamespace:
		return f.OperatorNamespace
	case LeaseNamespace:
		return f.LeaseNamespace
	case LogLevel:
		return f.LogLevel
	case LogFormat:
		return f.LogFormat
	case Debug:
		return strconv.FormatBool(f.Debug)
	case Trace:
		return strconv.FormatBool(f.Trace)
	}

	logger.Debugf("unknown flag value for option: %s", o)
	return ""
}

type ValueResolver struct {
	envVars       EnvVarsMap
	flagValues    *FlagValues
	hasConfigMap  bool
	configMapData map[string]string
}

func (vr ValueResolver) Get(o Option, defaultValue string) string {
	if val := vr.envVars[o.Env()]; val != "" {
		return val
	}

	if flagValue := vr.flagValues.ValueByOption(o); flagValue != "" {
		return flagValue
	}

	// Even if we could fetch all values via ConfigMap, some create chicken-and-egg issues.
	// So we will avoid them completely by never using ConfigMap for those values.
	if vr.hasConfigMap && o != OperatorNamespace {
		if configMapVal := o.ConfigMapValue(vr.configMapData); configMapVal != "" {
			return configMapVal
		}
	}

	return defaultValue
}

// OperatorSettings represents config values that the SCC Operator relies on to run
// These values are either set by: 1. Reading Env vars, or 2. the ConfigMap used by deployers
// This ensures that SCC Operator execution remains more uniform regardless of execution context.
type OperatorSettings struct {
	OperatorName    string
	Kubeconfig      string
	SystemNamespace string
	LeaseNamespace  string
	LogFormat       rootLog.Format
	LogLevel        logrus.Level
}

// Validate simply validates the configured settings are potentially valid but not if objects exist
func (s OperatorSettings) Validate() error {
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

func LoadInitialConfig(ctx context.Context, flags *FlagValues, envValues EnvVarsMap) (*OperatorSettings, error) {
	valueResolver := &ValueResolver{
		envVars:      envValues,
		flagValues:   flags,
		hasConfigMap: true,
	}

	restKubeConfig, err := kubeconfig.GetNonInteractiveClientConfig(flags.KubeconfigPath).ClientConfig()
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
		sccConfigMap = &corev1.ConfigMap{Data: make(map[string]string)}
		valueResolver.hasConfigMap = false
	}
	valueResolver.configMapData = sccConfigMap.Data

	loggingLevel := valueResolver.Get(LogLevel, "")
	trace, _ := strconv.ParseBool(valueResolver.Get(Trace, "false"))
	debug, _ := strconv.ParseBool(valueResolver.Get(Debug, "false"))

	loadedConfig := &OperatorSettings{
		Kubeconfig:      valueResolver.Get(KubeconfigPath, ""),
		OperatorName:    valueResolver.Get(OperatorName, consts.DefaultOperatorName),
		SystemNamespace: operatorNamespace,
		LeaseNamespace:  valueResolver.Get(LeaseNamespace, consts.DefaultLeaseNamespace),
		LogFormat:       decideLogFormat(valueResolver.Get(LogFormat, "")),
		LogLevel:        decideLogLevel(loggingLevel, trace, debug),
		// TODO: this is where we eventually add Dev mode and SCC mode settings too
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
