package config

import (
	"testing"

	"github.com/rancher/scc-operator/internal/config/option"
	"github.com/stretchr/testify/assert"
)

func TestValueResolver_SetConfigMapData(t *testing.T) {
	op := option.NewOption("vr-env", "", option.WithEnvKey("VR_ENV"))

	flags := option.Flags{}
	vr := &ValueResolver{
		envVars:      option.EnvVarsMap{"VR_ENV": "env-value"},
		flagValues:   &flags,
		hasConfigMap: false,
	}

	assert.Equal(t, "env-value", vr.Get(op, "default"))
	assert.Equal(t, false, vr.hasConfigMap)
	vr.SetConfigMapData(map[string]string{})
	assert.Equal(t, true, vr.hasConfigMap)
}

func TestValueResolver_EnvPrecedence(t *testing.T) {
	op := option.NewOption("vr-env", "", option.WithEnvKey("VR_ENV"))

	flags := option.Flags{}
	vr := &ValueResolver{
		envVars:      option.EnvVarsMap{"VR_ENV": "env-value"},
		flagValues:   &flags,
		hasConfigMap: false,
	}

	got := vr.Get(op, "default")
	assert.Equal(t, "env-value", got)
}

func TestValueResolver_DefaultFallback(t *testing.T) {
	op := option.NewOption("vr-default", "")

	flags := option.Flags{}
	vr := &ValueResolver{
		envVars:      option.EnvVarsMap{},
		flagValues:   &flags,
		hasConfigMap: false,
	}

	got := vr.Get(op, "the-default")
	assert.Equal(t, "the-default", got)
}

func TestValueResolver_ConfigMapAllowed(t *testing.T) {
	op := option.NewOption("vr-cm", "", option.WithConfigMapKey("cm-key"))

	flags := option.Flags{}
	vr := &ValueResolver{
		envVars:       option.EnvVarsMap{},
		flagValues:    &flags,
		hasConfigMap:  true,
		configMapData: map[string]string{"cm-key": "from-cm"},
	}

	got := vr.Get(op, "default")
	assert.Equal(t, "from-cm", got)
}

func TestValueResolver_FlagWhenDisabled(t *testing.T) {
	op := option.NewOption("vr-flag", "", option.WithoutFlag, option.WithFlagKey("flag-key"))

	flags := option.Flags{"flag-key": "from-flag"}
	vr := &ValueResolver{
		envVars:      option.EnvVarsMap{},
		flagValues:   &flags,
		hasConfigMap: false,
	}

	got := vr.Get(op, "default")
	assert.NotEqual(t, "from-flag", got)
	assert.Equal(t, "default", got)
}

func TestValueResolver_ConfigMapAllowedUnset(t *testing.T) {
	op := option.NewOption("vr-cm", "", option.WithConfigMapKey("cm-key"))

	flags := option.Flags{}
	vr := &ValueResolver{
		envVars:       option.EnvVarsMap{},
		flagValues:    &flags,
		hasConfigMap:  true,
		configMapData: map[string]string{},
	}

	got := vr.Get(op, "default")
	assert.Equal(t, "default", got)
}
