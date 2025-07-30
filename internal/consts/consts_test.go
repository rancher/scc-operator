package consts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSCCCredentialsSecretName(t *testing.T) {
	assert.Equal(t, "scc-system-credentials-", SCCCredentialsSecretName(""))
	assert.Equal(t, "scc-system-credentials-test", SCCCredentialsSecretName("test"))
}
