package option

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlags_Get(t *testing.T) {
	flags := Flags{
		"string-key": "a-string",
		"bool-key":   true,
		"int-key":    123,
	}

	t.Run("get string value", func(t *testing.T) {
		val, ok := flags.Get("string-key")
		assert.True(t, ok)
		assert.Equal(t, "a-string", val)
	})

	t.Run("get bool value", func(t *testing.T) {
		val, ok := flags.Get("bool-key")
		assert.True(t, ok)
		assert.Equal(t, "true", val)
	})

	t.Run("get int value", func(t *testing.T) {
		val, ok := flags.Get("int-key")
		assert.True(t, ok)
		assert.Equal(t, "123", val)
	})

	t.Run("get missing value", func(t *testing.T) {
		val, ok := flags.Get("missing-key")
		assert.False(t, ok)
		assert.Equal(t, "", val)
	})
}

func Test_AllFlags_Basic(t *testing.T) {
	// reset global registry
	options = make(map[string]RegisteredOption)

	// create options of different types
	strOpt := NewOption("string-opt", "default")
	intOpt := NewOption("int-opt", 42)
	boolOpt := NewOption("bool-opt", true)

	// set their FlagValue explicitly (simulating parsed flags)
	strOpt.FlagValue = "from-flag"
	intOpt.FlagValue = 7
	boolOpt.FlagValue = false

	flags := AllFlags()

	assert.Equal(t, 3, len(flags))
	assert.Equal(t, "from-flag", flags["string-opt"])
	assert.Equal(t, 7, flags["int-opt"])
	assert.Equal(t, false, flags["bool-opt"])

	// cleanup
	options = make(map[string]RegisteredOption)
	assert.Equal(t, 0, len(options))
}

func Test_AllEnvValues_Basic(t *testing.T) {
	// reset global registry
	options = make(map[string]RegisteredOption)

	// create options of different types
	strOpt := NewOption("string-opt", "default")
	intOpt := NewOption("int-opt", 42)
	boolOpt := NewOption("bool-opt", true, WithoutEnv)

	// Initial unconfigured test
	envVars := AllEnvValues()
	assert.Equal(t, 2, len(envVars))
	assert.Equal(t, "", envVars[strOpt.GetEnvKey()])
	assert.Equal(t, "", envVars[intOpt.GetEnvKey()])
	assert.Equal(t, "", envVars[boolOpt.GetEnvKey()])
	assert.Equal(t, "", boolOpt.GetEnv())

	configuredVars := ConfiguredEnvValues()
	assert.Equal(t, 0, len(configuredVars))

	// Configure some values
	t.Setenv(strOpt.GetEnvKey(), "some-value")
	t.Setenv(intOpt.GetEnvKey(), "25")

	envVars = AllEnvValues()
	assert.Equal(t, 2, len(envVars))
	assert.Equal(t, "some-value", envVars[strOpt.GetEnvKey()])
	assert.Equal(t, "25", envVars[intOpt.GetEnvKey()])
	assert.Equal(t, "", envVars[boolOpt.GetEnvKey()])

	configuredVars = ConfiguredEnvValues()
	assert.Equal(t, 2, len(configuredVars))

	// cleanup
	options = make(map[string]RegisteredOption)
	assert.Equal(t, 0, len(options))
}
