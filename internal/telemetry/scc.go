package telemetry

import (
	"github.com/SUSE/connect-ng/pkg/registration"
	"github.com/rancher/scc-operator/internal/rancher"
)

type subscriptionInfo struct {
	product string `json:"product"`
	version string `json:"version"`
	arch    string `json:"arch"`
	git     string `json:"git"`
}

type MetricsWrapper struct {
	Data             map[string]any
	subscriptionInfo subscriptionInfo
}

func NewMetricsWrapper(data map[string]any) MetricsWrapper {
	var subInfo subscriptionInfo
	subscriptionData := data["subscription"].(map[string]interface{})
	subInfo.product = subscriptionData["product"].(string)
	subInfo.version = subscriptionData["version"].(string)
	subInfo.arch = subscriptionData["arch"].(string)
	subInfo.git = subscriptionData["git"].(string)

	return MetricsWrapper{
		Data:             data,
		subscriptionInfo: subInfo,
	}
}

func (w *MetricsWrapper) ToSystemInformation() registration.SystemInformation {
	return w.Data
}

// GetProductIdentifier must return the SCC Product ID, the Product version, and product arch
func (w *MetricsWrapper) GetProductIdentifier() (string, string, string) {
	rancherVersion := rancher.Version(w.subscriptionInfo.version)
	return w.subscriptionInfo.product, rancherVersion.SCCSafeVersion(), w.subscriptionInfo.arch
}
