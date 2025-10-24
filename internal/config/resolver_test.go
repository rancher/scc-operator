package config

import (
	"testing"

	"github.com/rancher/scc-operator/internal/config/option"
	"github.com/stretchr/testify/assert"
)

func TestValueResolver_SetConfigMapData(t *testing.T) {
	op := option.NewOption("vr-env", "default", option.WithEnvKey("VR_ENV"))

	flags := option.Flags{}
	vr := &ValueResolver{
		envVars:      option.EnvVarsMap{"VR_ENV": "env-value"},
		flagValues:   &flags,
		hasConfigMap: false,
	}

	assert.Equal(t, "env-value", vr.Get(op))
	assert.Equal(t, false, vr.hasConfigMap)
	vr.SetConfigMapData(map[string]string{})
	assert.Equal(t, true, vr.hasConfigMap)
}

func TestValueResolver_EnvPrecedence(t *testing.T) {
	op := option.NewOption("vr-env", "default", option.WithEnvKey("VR_ENV"))

	flags := option.Flags{}
	vr := &ValueResolver{
		envVars:      option.EnvVarsMap{"VR_ENV": "env-value"},
		flagValues:   &flags,
		hasConfigMap: false,
	}

	assert.Equal(t, "env-value", vr.Get(op))
}

func TestValueResolver_DefaultFallback(t *testing.T) {
	op := option.NewOption("vr-default", "the-default")

	flags := option.Flags{}
	vr := &ValueResolver{
		envVars:      option.EnvVarsMap{},
		flagValues:   &flags,
		hasConfigMap: false,
	}

	assert.Equal(t, "the-default", vr.Get(op))
}

func TestValueResolver_ConfigMapAllowed(t *testing.T) {
	op := option.NewOption("vr-cm", "default", option.WithConfigMapKey("cm-key"))

	flags := option.Flags{}
	vr := &ValueResolver{
		envVars:       option.EnvVarsMap{},
		flagValues:    &flags,
		hasConfigMap:  true,
		configMapData: map[string]string{"cm-key": "from-cm"},
	}

	assert.Equal(t, "from-cm", vr.Get(op))
}

func TestValueResolver_FlagWhenDisabled(t *testing.T) {
	op := option.NewOption("vr-flag", "default", option.WithoutFlag, option.WithFlagKey("flag-key"))

	flags := option.Flags{"flag-key": "from-flag"}
	vr := &ValueResolver{
		envVars:      option.EnvVarsMap{},
		flagValues:   &flags,
		hasConfigMap: false,
	}

	got := vr.Get(op)
	assert.NotEqual(t, "from-flag", got)
	assert.Equal(t, "default", got)
}

func TestValueResolver_ConfigMapAllowedUnset(t *testing.T) {
	op := option.NewOption("vr-cm", "default", option.WithConfigMapKey("cm-key"))

	flags := option.Flags{}
	vr := &ValueResolver{
		envVars:       option.EnvVarsMap{},
		flagValues:    &flags,
		hasConfigMap:  true,
		configMapData: map[string]string{},
	}

	assert.Equal(t, "default", vr.Get(op))
}

func TestValueResolver_EmptyFlagFallsBack(t *testing.T) {
	op := option.NewOption("vr-empty-flag", "default", option.WithFlagKey("flag-key"))

	flags := option.Flags{"flag-key": ""}
	vr := &ValueResolver{
		envVars:      option.EnvVarsMap{},
		flagValues:   &flags,
		hasConfigMap: false,
	}

	assert.Equal(t, "default", vr.Get(op))
}
