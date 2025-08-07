package systeminfo

import (
	"github.com/google/uuid"
	"github.com/rancher/scc-operator/internal/rancher"
)

const (
	RancherProductIdentifier = "rancher"
	RancherCPUArch           = "unknown"
)

type InfoProvider struct {
	rancherUuid uuid.UUID
	clusterUuid uuid.UUID
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
	return rancher.GetServerURL()
}

func (i *InfoProvider) ServerHostname() string {
	serverHostname := rancher.ServerHostname()
	return serverHostname
}
