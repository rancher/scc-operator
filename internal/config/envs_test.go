package config

import "testing"

func TestEnvVarString(t *testing.T) {
	t.Parallel()
	cases := map[EnvVar]string{
		KubeconfigEnv:         "KUBECONFIG",
		LogLevelEnv:           "LOG_LEVEL",
		LogFormatEnv:          "LOG_FORMAT",
		SCCOperatorNameEnv:    "SCC_OPERATOR_NAME",
		SCCSystemNamespaceEnv: "SCC_SYSTEM_NAMESPACE",
		SCCLeaseNamespaceEnv:  "SCC_LEASE_NAMESPACE",
	}

	for k, want := range cases {
		t.Run(k.String(), func(t *testing.T) {
			t.Parallel()
			if got := k.String(); got != want {
				t.Fatalf("String() = %q, want %q", got, want)
			}
		})
	}
}

func TestEnvVarGet(t *testing.T) {
	cases := map[EnvVar]string{
		KubeconfigEnv:         "/tmp/kubeconfig",
		LogLevelEnv:           "debug",
		LogFormatEnv:          "json",
		SCCOperatorNameEnv:    "rancher-scc-operator",
		SCCSystemNamespaceEnv: "cattle-scc-system",
		SCCLeaseNamespaceEnv:  "cattle-lease",
	}

	for k, val := range cases {
		t.Run(k.String(), func(t *testing.T) {
			// t.Setenv ensures cleanup after the test.
			t.Setenv(k.String(), val)
			if got := k.Get(); got != val {
				t.Fatalf("Get() for %s = %q, want %q", k, got, val)
			}
		})
	}
}

func TestEnvVarsMapGet(t *testing.T) {
	t.Parallel()
	m := EnvVarsMap{
		KubeconfigEnv:         "/tmp/kubeconfig",
		LogLevelEnv:           "info",
		SCCSystemNamespaceEnv: "cattle-scc-system",
	}

	// Existing keys
	if got, want := m.Get(KubeconfigEnv), "/tmp/kubeconfig"; got != want {
		t.Fatalf("EnvVarsMap.Get(KubeconfigEnv) = %q, want %q", got, want)
	}
	if got, want := m.Get(LogLevelEnv), "info"; got != want {
		t.Fatalf("EnvVarsMap.Get(LogLevelEnv) = %q, want %q", got, want)
	}

	// Missing key should return zero value (empty string)
	if got := m.Get(LogFormatEnv); got != "" {
		t.Fatalf("EnvVarsMap.Get(LogFormatEnv) = %q, want empty string", got)
	}
}
