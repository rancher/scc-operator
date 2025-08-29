package telemetry

import (
	"strings"

	"github.com/SUSE/connect-ng/pkg/registration"
	"github.com/rancher/scc-operator/internal/rancher"
)

type subscriptionInfo struct {
	rancherUUID string
	product     string
	version     string
	arch        string
	git         string
}

type MetricsWrapper struct {
	Data             map[string]any
	subscriptionInfo subscriptionInfo
	rancherVersion   string
}

func NewMetricsWrapper(data map[string]any) MetricsWrapper {
	var subInfo subscriptionInfo
	subscriptionData := data["subscription"].(map[string]interface{})
	subInfo.rancherUUID = subscriptionData["installuuid"].(string)
	subInfo.product = subscriptionData["product"].(string)
	subInfo.version = subscriptionData["version"].(string)
	subInfo.arch = subscriptionData["arch"].(string)
	subInfo.git = subscriptionData["git"].(string)

	return MetricsWrapper{
		Data:             data,
		subscriptionInfo: subInfo,
		rancherVersion:   strings.TrimPrefix(subInfo.version, "v"),
	}
}

func (mw *MetricsWrapper) GetRancherUUID() string {
	return mw.subscriptionInfo.rancherUUID
}

func (mw *MetricsWrapper) ToSystemInformation() registration.SystemInformation {
	return mw.Data
}

// GetProductIdentifier must return the SCC Product ID, the Product version, and product arch
func (mw *MetricsWrapper) GetProductIdentifier() (string, string, string) {
	rancherVersion := rancher.Version(mw.rancherVersion)
	return mw.subscriptionInfo.product, rancherVersion.SCCSafeVersion(), mw.subscriptionInfo.arch
}
