package config

import (
	"github.com/rancher/scc-operator/internal/config/option"
	"github.com/spf13/pflag"
)

type ValueResolver struct {
	envVars       option.EnvVarsMap
	flagSet       *pflag.FlagSet
	hasConfigMap  bool
	configMapData map[string]string
}

func (vr *ValueResolver) SetConfigMapData(configMapData map[string]string) {
	vr.configMapData = configMapData
	vr.hasConfigMap = true
}

func (vr *ValueResolver) Get(o option.RegisteredOption) string {
	if val := vr.envVars[o.GetEnvKey()]; val != "" {
		return val
	}

	if o.AllowsFlag() && vr.flagSet != nil {
		flag := vr.flagSet.Lookup(o.GetFlagKey())
		if flag != nil && flag.Changed {
			// Changed can only be called on a parsed flag set.
			// It will return true if the flag has been set.
			// We can then get the value of the flag and return it.
			return flag.Value.String()
		}
	}

	if vr.hasConfigMap && o.AllowsConfigMap() {
		if configMapVal := vr.configMapData[o.GetConfigMapKey()]; configMapVal != "" {
			return configMapVal
		}
	}

	return o.GetDefaultAsString()
}

// NewValueResolver will prepare the collected flags and envs into a new value resolver
func NewValueResolver(flagSet *pflag.FlagSet) *ValueResolver {
	return &ValueResolver{
		envVars:      option.AllEnvValues(),
		flagSet:      flagSet,
		hasConfigMap: false,
	}
}
