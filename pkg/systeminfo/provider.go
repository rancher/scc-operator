package systeminfo

import (
	"github.com/google/uuid"
	v3 "github.com/rancher/scc-operator/internal/generated/controllers/management.cattle.io/v3"
	"github.com/rancher/scc-operator/internal/repos/settingrepo"
)

const (
	RancherProductIdentifier = "rancher"
	RancherCPUArch           = "unknown"
)

type InfoProvider struct {
	rancherUuid uuid.UUID
	clusterUuid uuid.UUID
	settings    *settingrepo.SettingRepository
	nodeCache   v3.NodeCache
}

// GetProductIdentifier returns a triple of product ID, version and CPU architecture
func (i *InfoProvider) GetProductIdentifier() (string, string, string) {
	// Rancher always returns "rancher" as product, and "unknown" as the architecture
	// The CPU architecture must match what SCC has product codes for; unless SCC adds other arches we always return unknown.
	// It is unlikely SCC should add these as that would require customers purchasing different RegCodes to run Rancher on arm64 and amd64.
	// In turn, that would lead to complications like "should Arm run Ranchers allow x86 downstream clusters?"
	return RancherProductIdentifier, SCCSafeVersion(), RancherCPUArch
}

// ServerUrl returns the Rancher server URL
func (i *InfoProvider) serverUrl() string {
	// Find setting from outside rancher
	return settingrepo.GetServerURL(i.settings)
}

func (i *InfoProvider) ServerHostname() string {
	serverHostname := settingrepo.ServerHostname(i.settings)
	return serverHostname
}
