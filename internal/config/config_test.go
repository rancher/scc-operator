package config

import (
	"testing"

	rootLog "github.com/rancher/scc-operator/internal/log"
	"github.com/sirupsen/logrus"
)

func TestOptionEnvMapping(t *testing.T) {
	t.Parallel()
	cases := map[Option]EnvVar{
		OperatorName:      SCCOperatorNameEnv,
		OperatorNamespace: SCCSystemNamespaceEnv,
		LeaseNamespace:    SCCLeaseNamespaceEnv,
		LogLevel:          LogLevelEnv,
		LogFormat:         LogFormatEnv,
		KubeconfigPath:    KubeconfigEnv,
		Debug:             DebugEnv,
		Trace:             TraceEnv,
	}

	for opt, want := range cases {
		t.Run(string(opt), func(t *testing.T) {
			t.Parallel()
			if got := opt.Env(); got != want {
				t.Fatalf("Env() for %q = %v, want %v", opt, got, want)
			}
		})
	}
}

func TestOptionConfigMapValue(t *testing.T) {
	t.Parallel()
	data := map[string]string{
		string(OperatorName):      "op-name",
		string(OperatorNamespace): "ns-ignored",
		string(LeaseNamespace):    "lease-ns",
		string(LogLevel):          "debug",
		string(LogFormat):         "json",
		string(KubeconfigPath):    "/path/kube",
		string(Debug):             "true",
		string(Trace):             "false",
	}

	// Existing keys
	if got, want := OperatorName.ConfigMapValue(data), "op-name"; got != want {
		t.Fatalf("OperatorName.ConfigMapValue = %q, want %q", got, want)
	}

	// Missing key returns empty string
	if got := Option("does-not-exist").ConfigMapValue(data); got != "" {
		t.Fatalf("Unknown option ConfigMapValue = %q, want empty", got)
	}
}

func TestFlagValuesValueByOption(t *testing.T) {
	t.Parallel()
	f := &FlagValues{
		KubeconfigPath:    "/tmp/kube",
		OperatorName:      "rancher-scc-operator",
		OperatorNamespace: "cattle-scc-system",
		LeaseNamespace:    "kube-system",
		LogLevel:          "warn",
		LogFormat:         "json",
		Debug:             true,
		Trace:             false,
	}

	if got, want := f.ValueByOption(KubeconfigPath), "/tmp/kube"; got != want {
		t.Fatalf("ValueByOption(KubeconfigPath) = %q, want %q", got, want)
	}
	if got, want := f.ValueByOption(OperatorName), "rancher-scc-operator"; got != want {
		t.Fatalf("ValueByOption(OperatorName) = %q, want %q", got, want)
	}
	if got, want := f.ValueByOption(OperatorNamespace), "cattle-scc-system"; got != want {
		t.Fatalf("ValueByOption(OperatorNamespace) = %q, want %q", got, want)
	}
	if got, want := f.ValueByOption(LeaseNamespace), "kube-system"; got != want {
		t.Fatalf("ValueByOption(LeaseNamespace) = %q, want %q", got, want)
	}
	if got, want := f.ValueByOption(LogLevel), "warn"; got != want {
		t.Fatalf("ValueByOption(LogLevel) = %q, want %q", got, want)
	}
	if got, want := f.ValueByOption(LogFormat), "json"; got != want {
		t.Fatalf("ValueByOption(LogFormat) = %q, want %q", got, want)
	}
	if got, want := f.ValueByOption(Debug), "true"; got != want {
		t.Fatalf("ValueByOption(Debug) = %q, want %q", got, want)
	}
	if got, want := f.ValueByOption(Trace), "false"; got != want {
		t.Fatalf("ValueByOption(Trace) = %q, want %q", got, want)
	}
}

func TestValueResolverPrecedence(t *testing.T) {
	t.Parallel()
	flags := &FlagValues{
		OperatorName:      "flag-op",
		OperatorNamespace: "flag-ns",
		LeaseNamespace:    "flag-lease",
		LogLevel:          "error",
	}
	vr := ValueResolver{
		envVars: EnvVarsMap{
			SCCOperatorNameEnv:    "env-op",
			SCCSystemNamespaceEnv: "env-ns",
			LogLevelEnv:           "trace", // should still be overridden by Trace bool later only in decideLogLevel. Here we just test precedence.
		},
		flagValues:    flags,
		hasConfigMap:  true,
		configMapData: map[string]string{string(OperatorName): "cm-op", string(OperatorNamespace): "cm-ns", string(LogLevel): "warn"},
	}

	// Env should beat flags and configmap
	if got, want := vr.Get(OperatorName, "def-op"), "env-op"; got != want {
		t.Fatalf("ValueResolver.Get OperatorName = %q, want %q", got, want)
	}

	// OperatorNamespace should ignore configmap and prefer env, then flags, then default
	// Here env is set, so it should return env value
	if got, want := vr.Get(OperatorNamespace, "def-ns"), "env-ns"; got != want {
		t.Fatalf("ValueResolver.Get OperatorNamespace = %q, want %q", got, want)
	}

	// For a key without env value, it should fall back to flag
	if got, want := vr.Get(LeaseNamespace, "def-lease"), "flag-lease"; got != want {
		t.Fatalf("ValueResolver.Get LeaseNamespace = %q, want %q", got, want)
	}

	// For a key without env or flag, and not OperatorNamespace, it should use ConfigMap
	vrNoEnvFlag := ValueResolver{
		envVars:       EnvVarsMap{},
		flagValues:    &FlagValues{},
		hasConfigMap:  true,
		configMapData: map[string]string{string(LogLevel): "debug"},
	}
	if got, want := vrNoEnvFlag.Get(LogLevel, "info"), "debug"; got != want {
		t.Fatalf("ValueResolver.Get(LogLevel) with only CM = %q, want %q", got, want)
	}

	// OperatorNamespace should NOT read from ConfigMap, so default should be used
	vrOpNS := ValueResolver{
		envVars:       EnvVarsMap{},
		flagValues:    &FlagValues{},
		hasConfigMap:  true,
		configMapData: map[string]string{string(OperatorNamespace): "cm-ns"},
	}
	if got, want := vrOpNS.Get(OperatorNamespace, "default-ns"), "default-ns"; got != want {
		t.Fatalf("ValueResolver.Get(OperatorNamespace) should ignore CM. got %q, want %q", got, want)
	}
}

func TestDecideLogFormat(t *testing.T) {
	t.Parallel()
	// Valid format
	if got := decideLogFormat("json"); got != rootLog.FormatJSON {
		t.Fatalf("decideLogFormat(json) = %v, want %v", got, rootLog.FormatJSON)
	}

	// Invalid should default
	if got := decideLogFormat("yaml"); got != rootLog.DefaultFormat {
		t.Fatalf("decideLogFormat(yaml) = %v, want default %v", got, rootLog.DefaultFormat)
	}
}

func TestDecideLogLevel(t *testing.T) {
	t.Parallel()
	if got := decideLogLevel("info", true, false); got != logrus.TraceLevel {
		t.Fatalf("decideLogLevel(trace) = %v, want TraceLevel", got)
	}
	if got := decideLogLevel("info", false, true); got != logrus.DebugLevel {
		t.Fatalf("decideLogLevel(debug) = %v, want DebugLevel", got)
	}
	if got := decideLogLevel("error", false, false); got != logrus.ErrorLevel {
		t.Fatalf("decideLogLevel(error) = %v, want ErrorLevel", got)
	}
	if got := decideLogLevel("not-a-level", false, false); got != logrus.InfoLevel {
		t.Fatalf("decideLogLevel(invalid) = %v, want InfoLevel", got)
	}
}

func TestOperatorSettingsValidate(t *testing.T) {
	t.Parallel()
	// Missing operator name
	s := OperatorSettings{}
	if err := s.Validate(); err == nil {
		t.Fatal("Validate() expected error for missing OperatorName, got nil")
	}

	// Missing system namespace
	s.OperatorName = "op"
	if err := s.Validate(); err == nil {
		t.Fatal("Validate() expected error for missing SystemNamespace, got nil")
	}

	// Empty lease namespace should not error (only logs a warning)
	s.SystemNamespace = "scc-ns"
	s.LeaseNamespace = ""
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error with empty LeaseNamespace: %v", err)
	}

	// With all fields set
	s.LeaseNamespace = "kube-system"
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}
}
