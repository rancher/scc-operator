package systeminfo

import (
	"github.com/google/uuid"
	v3 "github.com/rancher/scc-operator/internal/generated/controllers/management.cattle.io/v3"
	"github.com/rancher/scc-operator/internal/repos/settingrepo"
	"k8s.io/apimachinery/pkg/labels"
	//"github.com/rancher/rancher/pkg/settings"
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

func NewInfoProvider(
	settings *settingrepo.SettingRepository,
	nodeCache v3.NodeCache,
) *InfoProvider {
	return &InfoProvider{
		settings:  settings,
		nodeCache: nodeCache,
	}
}

func (i *InfoProvider) SetUuids(rancherUuid uuid.UUID, clusterUuid uuid.UUID) *InfoProvider {
	i.rancherUuid = rancherUuid
	i.clusterUuid = clusterUuid
	return i
}

// GetProductIdentifier returns a triple of product ID, version and CPU architecture
func (i *InfoProvider) GetProductIdentifier() (string, string, string) {
	// Rancher always returns "rancher" as product, and "unknown" as the architecture
	// The CPU architecture must match what SCC has product codes for; unless SCC adds other arches we always return unknown.
	// It is unlikely SCC should add these as that would require customers purchasing different RegCodes to run Rancher on arm64 and amd64.
	// In turn, that would lead to complications like "should Arm run Ranchers allow x86 downstream clusters?"
	return RancherProductIdentifier, SCCSafeVersion(), RancherCPUArch
}

func (i *InfoProvider) IsLocalReady() bool {
	localNodes, nodesErr := i.nodeCache.List("local", labels.Everything())
	// TODO: should this also check status of nodes and only count ready/healthy nodes?
	if nodesErr == nil && len(localNodes) > 0 {
		return true
	}

	return false
}

// CanStartSccOperator determines when the SCC operator should fully start
// Currently this waits for a valid Server URL to be configured and the local cluster to appear ready
func (i *InfoProvider) CanStartSccOperator() bool {
	return i.IsServerUrlReady() && i.IsLocalReady()
}

// ServerUrl returns the Rancher server URL
func (i *InfoProvider) serverUrl() string {
	// Find setting from outside rancher
	return settingrepo.GetServerURL(i.settings)
}

func (i *InfoProvider) IsServerUrlReady() bool {
	serverUrl := i.serverUrl()
	return serverUrl != ""
}

func (i *InfoProvider) ServerHostname() string {
	serverHostname := settingrepo.ServerHostname(i.settings)
	return serverHostname
}
