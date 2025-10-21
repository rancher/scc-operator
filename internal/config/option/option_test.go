package option

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Never add t.Parallel() to this one...
func Test_NewOption_Basic(t *testing.T) {
	var DebugSetting = NewOption("debug", false)

	assert.Equal(t, DebugSetting.Name, "debug")
	assert.False(t, DebugSetting.Default)
	assert.Equal(t, "false", DebugSetting.GetDefaultAsString())
	assert.Equal(t, DebugSetting.AllowFromConfigMap, false)

	// asserts about the global repo
	assert.Equal(t, len(options), 1)
	assert.Equal(t, len(AllOptions()), 1)

	var ConfigMapToggleable = NewOption("configmap-toggleable", "", AllowedFromConfigMap)
	assert.Equal(t, ConfigMapToggleable.Name, "configmap-toggleable")
	assert.Equal(t, ConfigMapToggleable.Default, "")
	assert.Equal(t, ConfigMapToggleable.AllowFromConfigMap, true)
	assert.Equal(t, ConfigMapToggleable.ConfigMapKey, "configmap-toggleable")
	assert.Equal(t, len(options), 2)
	assert.Equal(t, len(AllOptions()), 2)

	options = make(map[string]RegisteredOption)
	assert.Equal(t, len(options), 0)
	assert.Equal(t, len(AllOptions()), 0)
}

func Test_NewOption_GetDefaultAsString(t *testing.T) {
	var BoolSetting = NewOption("debug", false)
	assert.Equal(t, "false", BoolSetting.GetDefaultAsString())

	var IntSetting = NewOption("timeout", 60)
	assert.Equal(t, "60", IntSetting.GetDefaultAsString())

	var FloatSetting = NewOption("speed", 60.0)
	assert.Equal(t, "60.00", FloatSetting.GetDefaultAsString())

	var StringSetting = NewOption("mode", "default")
	assert.Equal(t, "default", StringSetting.GetDefaultAsString())
}

func Test_NewOption_DefaultKeys(t *testing.T) {
	options = make(map[string]RegisteredOption)

	opt := NewOption("some-option", 123)

	assert.Equal(t, "some-option", opt.Name)
	assert.Equal(t, 123, opt.Default)
	assert.True(t, opt.AllowFromEnv)
	assert.True(t, opt.AllowFromFlag)
	assert.False(t, opt.AllowFromConfigMap)

	// default keys should be auto-populated for env and flag
	assert.Equal(t, "SOME_OPTION", opt.EnvKey)
	assert.Equal(t, "some-option", opt.FlagKey)
	// configmap not allowed by default, so key should be empty
	assert.Equal(t, "", opt.ConfigMapKey)

	// ensure the global registry tracked it
	assert.Equal(t, 1, len(options))

	options = make(map[string]RegisteredOption)
	assert.Equal(t, 0, len(options))
}

func Test_NewOption_CustomKeys(t *testing.T) {
	options = make(map[string]RegisteredOption)

	opt := NewOption("another-opt", true, WithEnvKey("CUSTOM_ENV"), WithFlagKey("custom-flag"))
	assert.Equal(t, "another-opt", opt.Name)
	assert.Equal(t, true, opt.Default)
	assert.Equal(t, "CUSTOM_ENV", opt.EnvKey)
	assert.Equal(t, "custom-flag", opt.FlagKey)
	assert.True(t, opt.AllowFromEnv)
	assert.True(t, opt.AllowFromFlag)
	assert.False(t, opt.AllowFromConfigMap)

	// WithConfigMapKey should set the key and allow from configmap
	cm := NewOption("with-cm", "val", WithConfigMapKey("my-key"))
	assert.True(t, cm.AllowFromConfigMap)
	assert.Equal(t, "my-key", cm.ConfigMapKey)

	// registry should have both
	assert.Equal(t, 2, len(options))

	options = make(map[string]RegisteredOption)
	assert.Equal(t, 0, len(options))
}

func Test_NewOption_WithoutEnvAndFlag(t *testing.T) {
	options = make(map[string]RegisteredOption)

	opt := NewOption("no-io", 0, WithoutEnv, WithoutFlag)

	assert.Equal(t, "no-io", opt.Name)
	// when disabled, keys should remain empty and not auto-populated
	assert.Equal(t, "", opt.EnvKey)
	assert.Equal(t, "", opt.FlagKey)
	assert.False(t, opt.AllowFromEnv)
	assert.False(t, opt.AllowFromFlag)

	// ConfigMap still disabled by default, no key
	assert.False(t, opt.AllowFromConfigMap)
	assert.Equal(t, "", opt.ConfigMapKey)

	assert.Equal(t, 1, len(options))

	options = make(map[string]RegisteredOption)
	assert.Equal(t, 0, len(options))
}

func Test_Option_TypeMethod(t *testing.T) {
	options = make(map[string]RegisteredOption)

	optInt := NewOption("int-opt", 0)
	optStr := NewOption("str-opt", "")
	optBool := NewOption("bool-opt", false)

	assert.Equal(t, reflect.TypeOf(0), optInt.Type())
	assert.Equal(t, reflect.TypeOf(""), optStr.Type())
	assert.Equal(t, reflect.TypeOf(false), optBool.Type())

	options = make(map[string]RegisteredOption)
	assert.Equal(t, 0, len(options))
}
