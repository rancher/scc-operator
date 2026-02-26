package telemetry

import (
	"github.com/SUSE/connect-ng/pkg/registration"
	"github.com/rancher/scc-operator/internal/semver"
)

type subscriptionInfo struct {
	rancherUUID  string
	product      string
	buildVersion string
	arch         string
	git          string
}

type MetricsWrapper struct {
	Data             map[string]any
	productVersion   string
	subscriptionInfo subscriptionInfo
}

func NewMetricsWrapper(data map[string]any) MetricsWrapper {
	var subInfo subscriptionInfo
	subscriptionData := data["subscription"].(map[string]interface{})
	subInfo.rancherUUID = subscriptionData["installuuid"].(string)
	subInfo.product = subscriptionData["product"].(string)
	subInfo.buildVersion = subscriptionData["version"].(string)
	subInfo.arch = subscriptionData["arch"].(string)
	subInfo.git = subscriptionData["git"].(string)

	return MetricsWrapper{
		Data:             data,
		productVersion:   data["version"].(string),
		subscriptionInfo: subInfo,
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
	rancherVersion := semver.Version(mw.productVersion)
	return mw.subscriptionInfo.product, rancherVersion.SCCSafeVersion(), mw.subscriptionInfo.arch
}
