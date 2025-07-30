package systeminfo

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/rancher/scc-operator/internal/telemetry"
	"github.com/rancher/scc-operator/internal/telemetry/secret"

	//"k8s.io/client-go/util/retry"
	"github.com/SUSE/connect-ng/pkg/registration"
	//"github.com/rancher/rancher/pkg/telemetry"
	rootLog "github.com/rancher/scc-operator/internal/log"
	//"github.com/rancher/scc-operator/pkg/systeminfo/secret"
	"github.com/rancher/scc-operator/pkg/util"
)

type InfoExporter struct {
	infoProvider         *InfoProvider
	tel                  telemetry.TelemetryGatherer
	isLocalReady         bool
	logger               rootLog.StructuredLogger
	metricsSecretManager *secret.MetricsSecretManager
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

func NewInfoExporter(
	infoProvider *InfoProvider,
	rancherTelemetry telemetry.TelemetryGatherer,
	logger rootLog.StructuredLogger,
	metricsSecretManager *secret.MetricsSecretManager,
) *InfoExporter {
	return &InfoExporter{
		infoProvider:         infoProvider,
		tel:                  rancherTelemetry,
		isLocalReady:         false,
		logger:               logger,
		metricsSecretManager: metricsSecretManager,
	}
}

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

func (e *InfoExporter) ClusterUuid() uuid.UUID {
	return e.infoProvider.clusterUuid
}

func (e *InfoExporter) preparedForSCC() RancherSCCInfo {
	/*
		var exporter telemetry.RancherManagerTelemetry
		// TODO(dan): this logic might need some tweaking
		if err := retry.OnError(retry.DefaultRetry, func(_ error) bool {
			return true
		}, func() error {
			exp, err := e.tel.GetClusterTelemetry()
			if err != nil {
				return err
			}
			exporter = exp
			return nil
		}); err != nil {
			// TODO(dan) : should probably surface an error here and handle it
			return RancherSCCInfo{}
		}

		nodeCount := 0
		totalCpuCores := int(0)
		// note: this will only correctly report up to ~9 exabytes of memory,
		// which should be fine
		totalMemBytes := int(0)
		clusterCount := exporter.ManagedClusterCount()

		// local cluster metrics
		localClT := exporter.LocalClusterTelemetry()
		localCores, _ := localClT.CpuCores()
		localMem, _ := localClT.MemoryCapacityBytes()
		totalCpuCores += localCores
		totalMemBytes += localMem
		for _, _ = range localClT.PerNodeTelemetry() {
			nodeCount++
		}

		// managed cluster metrics
		for _, clT := range exporter.PerManagedClusterTelemetry() {
			cores, _ := clT.CpuCores()
			totalCpuCores += cores
			memBytes, _ := clT.MemoryCapacityBytes()
			totalMemBytes += memBytes
			for _, _ = range clT.PerNodeTelemetry() {
				nodeCount++
			}
		}
	*/

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

	metricsUpdateErr := e.metricsSecretManager.UpdateMetricsDebugSecret(sccJson)
	if metricsUpdateErr != nil {
		e.logger.Errorf("error updating metrics secret: %v", metricsUpdateErr)
	}

	systemInfoMap := make(registration.SystemInformation)
	err := json.Unmarshal(sccJson, &systemInfoMap)
	if err != nil {
		return nil, err
	}

	return systemInfoMap, nil
}
