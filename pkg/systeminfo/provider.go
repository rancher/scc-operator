package systeminfo

import (
	"github.com/google/uuid"
	//"k8s.io/apimachinery/pkg/labels"
	"net/url"
	//v3 "github.com/rancher/rancher/pkg/generated/controllers/management.cattle.io/v3"
	//"github.com/rancher/rancher/pkg/settings"
)

const (
	RancherProductIdentifier = "rancher"
	RancherCPUArch           = "unknown"
)

type InfoProvider struct {
	rancherUuid uuid.UUID
	clusterUuid uuid.UUID
	//nodeCache   v3.NodeCache
}

func NewInfoProvider(
// nodeCache v3.NodeCache,
) *InfoProvider {
	return &InfoProvider{
		//nodeCache: nodeCache,
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
	//localNodes, nodesErr := i.nodeCache.List("local", labels.Everything())
	// TODO: should this also check status of nodes and only count ready/healthy nodes?
	//if nodesErr == nil && len(localNodes) > 0 {
	if true {
		return true
	}

	return false
}

// CanStartSccOperator determines when the SCC operator should fully start
// Currently this waits for a valid Server URL to be configured and the local cluster to appear ready
func (i *InfoProvider) CanStartSccOperator() bool {
	return IsServerUrlReady() && i.IsLocalReady()
}

// ServerUrl returns the Rancher server URL
func ServerUrl() string {
	// Find setting from outside rancher
	//	return settings.ServerURL.Get()
	return "localhost:8080"
}

// ServerHostname returns the hostname of the Rancher server URL
func ServerHostname() string {
	serverUrl := ServerUrl()
	if serverUrl == "" {
		return ""
	}
	parsed, _ := url.Parse(serverUrl)
	return parsed.Host
}

func IsServerUrlReady() bool {
	return ServerHostname() != ""
}
