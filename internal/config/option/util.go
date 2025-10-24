package option

import (
	"fmt"
	"reflect"
)

func AllOptions() map[string]RegisteredOption {
	return options
}

type Flags map[string]any

func (f Flags) Get(key string) (string, bool) {
	value, ok := f[key]
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%v", value), ok
}

func AllFlags() Flags {
	flags := make(Flags)

	for _, ro := range options {
		// Use the option's name as the key, and reflect to get the generic FlagValue
		rv := reflect.ValueOf(ro)
		if rv.Kind() == reflect.Pointer {
			rv = rv.Elem()
		}
		if rv.IsValid() {
			fv := rv.FieldByName("FlagValue")
			if fv.IsValid() {
				flags[ro.GetName()] = fv.Interface()
			}
		}
	}

	return flags
}

type EnvVarsMap map[string]string

func AllEnvValues() EnvVarsMap {
	envMap := make(EnvVarsMap)

	for _, registeredOption := range options {
		if registeredOption.AllowsEnv() {
			envMap[registeredOption.GetEnvKey()] = registeredOption.GetEnv()
		}
	}

	return envMap
}

func ConfiguredEnvValues() EnvVarsMap {
	envMap := make(EnvVarsMap)

	for _, registeredOption := range options {
		if registeredOption.AllowsEnv() {
			if envVal := registeredOption.GetEnv(); envVal != "" {
				envMap[registeredOption.GetEnvKey()] = envVal
			}
		}
	}

	return envMap
}
