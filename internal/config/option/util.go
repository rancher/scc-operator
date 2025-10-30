package option

func AllOptions() map[string]RegisteredOption {
	return options
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
