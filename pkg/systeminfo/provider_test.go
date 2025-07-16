package systeminfo

import (
	"github.com/google/uuid"
	"github.com/rancher-sandbox/scc-operator/internal/settings"
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

func TestGetProductIdentifier(t *testing.T) {
	randUuid := uuid.New()
	rancherUuid, _ := uuid.Parse(randUuid.String())
	clusterUuid, _ := uuid.Parse(uuid.New().String())

	infoProvider := NewInfoProvider().SetUuids(rancherUuid, clusterUuid)
	product, version, architecture := infoProvider.GetProductIdentifier()
	assert.Equal(t, "rancher", product)
	// When in dev mode, the info provider has to "lie" in order to connect with SCC
	// however, when not in dev mode, the info provider should return the correct version
	if versionIsDevBuild() {
		assert.NotEqual(t, coreRancherVersion, version)
	} else {
		assert.Equal(t, coreRancherVersion, version)
	}
	assert.Equal(t, SCCSafeVersion(), version)
	assert.Equal(t, "unknown", architecture)
}

func TestServerHostname(t *testing.T) {
	originalUrl := settings.GetServerURL()
	assert.IsType(t, string(""), originalUrl)
}
