package systeminfo

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/SUSE/connect-ng/pkg/registration"
	rootLog "github.com/rancher/scc-operator/internal/log"
	"github.com/rancher/scc-operator/pkg/util"
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

// GetProductIdentifier returns a triple of product ID, version and CPU architecture
func (e *InfoExporter) GetProductIdentifier() (string, string, string) {
	return e.infoProvider.GetProductIdentifier()
}

func (e *InfoExporter) RancherUuid() uuid.UUID {
	return e.infoProvider.rancherUuid
}

func (e *InfoExporter) preparedForSCC() RancherSCCInfo {

	return RancherSCCInfo{
		UUID:             e.infoProvider.rancherUuid,
		RancherUrl:       e.infoProvider.serverUrl(),
		Version:          "unknown",
		Nodes:            1,
		Sockets:          0,
		Clusters:         1,
		CpuCores:         1,
		MemoryBytesTotal: util.BytesToMiBRounded(20_000),
	}
}

func (e *InfoExporter) PreparedForSCC() (registration.SystemInformation, error) {
	sccPreparedInfo := e.preparedForSCC()
	sccJson, jsonErr := json.Marshal(sccPreparedInfo)
	if jsonErr != nil {
		return nil, jsonErr
	}

	systemInfoMap := make(registration.SystemInformation)
	err := json.Unmarshal(sccJson, &systemInfoMap)
	if err != nil {
		return nil, err
	}

	return systemInfoMap, nil
}
