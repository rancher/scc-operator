package systeminfo

import (
	"github.com/google/uuid"
)

const (
	RancherProductIdentifier = "rancher"
	RancherCPUArch           = "unknown"
)

type InfoProvider struct {
	rancherUuid uuid.UUID
	clusterUuid uuid.UUID
}

// ServerUrl returns the Rancher server URL
func (i *InfoProvider) serverUrl() string {
	// TODO: replace
	return "TODO: repalce"
}

func (i *InfoProvider) ServerHostname() string {
	// TODO: replace
	return "TODO: repalce"
}
