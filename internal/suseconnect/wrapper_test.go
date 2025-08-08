package suseconnect

import (
	"testing"

	"github.com/SUSE/connect-ng/pkg/connection"
	"github.com/stretchr/testify/assert"
)

func TestDefaultConnectionOptionsBasic(t *testing.T) {
	defaultOptions := DefaultConnectionOptions("rancher-scc-integration", "0.0.1")
	expected := connection.Options{
		URL:              connection.DefaultBaseURL,
		Secure:           true,
		AppName:          "rancher-scc-integration",
		Version:          "0.0.1",
		PreferedLanguage: "en_US",
		Timeout:          connection.DefaultTimeout,
	}
	assert.Equal(t, expected, defaultOptions)
}

func TestDefaultConnectionOptions(t *testing.T) {
	defaultOptions := DefaultConnectionOptions("rancher-scc-integration", "0.0.1")
	assert.Equal(t, connection.DefaultBaseURL, defaultOptions.URL)
	assert.Equal(t, "rancher-scc-integration", defaultOptions.AppName)
	assert.Equal(t, "0.0.1", defaultOptions.Version)
}

func TestDefaultRancherConnection(t *testing.T) {
	//Options := DefaultConnectionOptions()
	//expected := connection.New(Options, connection.NoCredentials{})

	//assert.Equal(t, expected, DefaultRancherConnection(connection.NoCredentials{}))
}
