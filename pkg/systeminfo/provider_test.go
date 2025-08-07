package systeminfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewInfoProvider(t *testing.T) {
	// Test with dev build
	// infoProvider := NewInfoProvider(rancherUuid, clusterUuid)
	assert.Equal(t, "other", SCCSafeVersion())

	// Test with non-dev build
	coreRancherVersion = "1.9.0"
	defer func() { coreRancherVersion = "dev" }()
	// infoProvider = NewInfoProvider(rancherUuid, clusterUuid)
	assert.Equal(t, "1.9.0", SCCSafeVersion())

	// Test with no mock version
	// infoProvider = NewInfoProvider(rancherUuid, clusterUuid)
	assert.Equal(t, coreRancherVersion, SCCSafeVersion())
}
