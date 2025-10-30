package option

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
