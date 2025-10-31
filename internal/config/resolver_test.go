package config

import (
	"testing"

	"github.com/rancher/scc-operator/internal/config/option"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestValueResolver_SetConfigMapData(t *testing.T) {
	op := option.NewOption("vr-env", "default", option.WithEnvKey("VR_ENV"))

	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	vr := &ValueResolver{
		envVars:      option.EnvVarsMap{"VR_ENV": "env-value"},
		flagSet:      flagSet,
		hasConfigMap: false,
	}

	assert.Equal(t, "env-value", vr.Get(op))
	assert.Equal(t, false, vr.hasConfigMap)
	vr.SetConfigMapData(map[string]string{})
	assert.Equal(t, true, vr.hasConfigMap)
}

func TestValueResolver_EnvPrecedence(t *testing.T) {
	op := option.NewOption("vr-env", "default", option.WithEnvKey("VR_ENV"))

	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	vr := &ValueResolver{
		envVars:      option.EnvVarsMap{"VR_ENV": "env-value"},
		flagSet:      flagSet,
		hasConfigMap: false,
	}

	assert.Equal(t, "env-value", vr.Get(op))
}

func TestValueResolver_DefaultFallback(t *testing.T) {
	op := option.NewOption("vr-default", "the-default")

	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	vr := &ValueResolver{
		envVars:      option.EnvVarsMap{},
		flagSet:      flagSet,
		hasConfigMap: false,
	}

	assert.Equal(t, "the-default", vr.Get(op))
}

func TestValueResolver_ConfigMapAllowed(t *testing.T) {
	op := option.NewOption("vr-cm", "default", option.WithConfigMapKey("cm-key"))

	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	vr := &ValueResolver{
		envVars:       option.EnvVarsMap{},
		flagSet:       flagSet,
		hasConfigMap:  true,
		configMapData: map[string]string{"cm-key": "from-cm"},
	}

	assert.Equal(t, "from-cm", vr.Get(op))
}

func TestValueResolver_FlagWhenDisabled(t *testing.T) {
	op := option.NewOption("vr-flag", "default", option.WithoutFlag, option.WithFlagKey("flag-key"))

	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String("flag-key", "from-flag", "")
	flagSet.Parse([]string{"--flag-key=from-flag"})

	vr := &ValueResolver{
		envVars:      option.EnvVarsMap{},
		flagSet:      flagSet,
		hasConfigMap: false,
	}

	got := vr.Get(op)
	assert.NotEqual(t, "from-flag", got)
	assert.Equal(t, "default", got)
}

func TestValueResolver_ConfigMapAllowedUnset(t *testing.T) {
	op := option.NewOption("vr-cm", "default", option.WithConfigMapKey("cm-key"))

	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	vr := &ValueResolver{
		envVars:       option.EnvVarsMap{},
		flagSet:       flagSet,
		hasConfigMap:  true,
		configMapData: map[string]string{},
	}

	assert.Equal(t, "default", vr.Get(op))
}

func TestValueResolver_EmptyFlagIsHonored(t *testing.T) {
	op := option.NewOption("vr-empty-flag", "default", option.WithFlagKey("flag-key"))

	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String("flag-key", "default", "")
	flagSet.Parse([]string{"--flag-key="})

	vr := &ValueResolver{
		envVars:      option.EnvVarsMap{},
		flagSet:      flagSet,
		hasConfigMap: false,
	}

	assert.Equal(t, "", vr.Get(op))
}

func TestValueResolver_DefaultFlagDoesNotOverrideConfigMap(t *testing.T) {
	op := option.NewOption("debug", false, option.AllowedFromConfigMap)

	// Simulate flags being parsed, where "debug" has a default value of false.
	// The flag default is then parsed into the `option.Flags` as a string value
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.Bool("debug", false, "A test bool flag")
	_ = flagSet.Parse([]string{}) // not passed. Changed is false.

	vr := &ValueResolver{
		envVars:       option.EnvVarsMap{},
		flagSet:       flagSet,
		hasConfigMap:  true,
		configMapData: map[string]string{"debug": "true"},
	}

	// The bug is that the flag's default value ("flag-default-value") will be returned,
	// instead of the value from the config map ("cm-value").
	// This test will fail with the old code.
	assert.Equal(t, "true", vr.Get(op))
}
