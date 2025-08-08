package systeminfo

import (
	"github.com/google/uuid"

	"github.com/SUSE/connect-ng/pkg/registration"
	rootLog "github.com/rancher/scc-operator/internal/log"
)

type InfoExporter struct {
	infoProvider *InfoProvider
	isLocalReady bool
	logger       rootLog.StructuredLogger
}

type RancherSCCInfo struct {
	UUID             uuid.UUID `json:"uuid"`
	RancherUrl       string    `json:"server_url"`
	Nodes            int       `json:"nodes"`
	Sockets          int       `json:"sockets"`
	Clusters         int       `json:"clusters"`
	Version          string    `json:"version"`
	CpuCores         int       `json:"vcpus"`
	MemoryBytesTotal int       `json:"mem_total"`
}

type ProductTriplet struct {
	Identifier string `json:"identifier"`
	Version    string `json:"version"`
	Arch       string `json:"arch"`
}

type RancherOfflineRequest struct {
	Product ProductTriplet `json:"product"`

	UUID       uuid.UUID `json:"uuid"`
	RancherUrl string    `json:"server_url"`
}

type RancherOfflineRequestEncoded []byte

func (e *InfoExporter) Provider() *InfoProvider {
	return e.infoProvider
}

func (e *InfoExporter) RancherUuid() uuid.UUID {
	return e.infoProvider.rancherUuid
}

func (e *InfoExporter) PreparedForSCC() (registration.SystemInformation, error) {
	// TODO cleanup

	return nil, nil
}
