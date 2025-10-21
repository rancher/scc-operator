package config

import "github.com/rancher/scc-operator/internal/config/option"

type ValueResolver struct {
	envVars       option.EnvVarsMap
	flagValues    *option.Flags
	hasConfigMap  bool
	configMapData map[string]string
}

func (vr ValueResolver) Get(o option.RegisteredOption, defaultValue string) string {
	if val := vr.envVars[o.GetEnvKey()]; val != "" {
		return val
	}

	if !o.AllowsFlag() {
		if flagValue, hasFlagValue := vr.flagValues.Get(o.GetFlagKey()); hasFlagValue {
			return flagValue
		}
	}

	if vr.hasConfigMap && o.AllowsConfigMap() {
		if configMapVal := vr.configMapData[o.GetConfigMapKey()]; configMapVal != "" {
			return configMapVal
		}
	}

	return defaultValue
}

// NewValueResolver will prepare the collected flags and envs into a new value resolver
func NewValueResolver() *ValueResolver {
	flags := option.AllFlags()
	return &ValueResolver{
		envVars:      option.AllEnvValues(),
		flagValues:   &flags,
		hasConfigMap: false,
	}
}
