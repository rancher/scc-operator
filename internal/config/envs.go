package config

import (
	"os"
	"strconv"
)

type EnvVar string

// Note: We need two debug/trace Envs to match ranchers 2 Envs
const (
	KubeconfigEnv         EnvVar = "KUBECONFIG"
	SCCOperatorNameEnv    EnvVar = "SCC_OPERATOR_NAME"
	SCCSystemNamespaceEnv EnvVar = "SCC_SYSTEM_NAMESPACE"
	SCCLeaseNamespaceEnv  EnvVar = "SCC_LEASE_NAMESPACE"
	LogFormatEnv          EnvVar = "LOG_FORMAT"
	LogLevelEnv           EnvVar = "LOG_LEVEL"

	DebugEnv  EnvVar = "CATTLE_DEBUG"
	Debug2Env EnvVar = "RANCHER_DEBUG"

	TraceEnv  EnvVar = "CATTLE_TRACE"
	Trace2Env EnvVar = "RANCHER_TRACE"
)

func (k EnvVar) String() string {
	return string(k)
}

func (k EnvVar) Get() string {
	return os.Getenv(k.String())
}

func (k EnvVar) IsEnabled() bool {
	value := k.Get()
	if value == "" {
		return false
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}

	return boolValue
}

func GetDebugEnv() string {
	return strconv.FormatBool(DebugEnv.IsEnabled() || Debug2Env.IsEnabled())
}

func GetTraceEnv() string {
	return strconv.FormatBool(TraceEnv.IsEnabled() || Trace2Env.IsEnabled())
}

type EnvVarsMap map[EnvVar]string

func (em EnvVarsMap) Get(name EnvVar) string {
	return em[name]
}
