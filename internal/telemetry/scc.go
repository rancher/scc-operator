package telemetry

import (
	"github.com/SUSE/connect-ng/pkg/registration"
	"github.com/rancher/scc-operator/internal/semver"
	"github.com/rancher/scc-operator/internal/suseconnect/products"
)

type MetricsWrapper interface {
	// GetSystemUUID returns the UUID of the system/cluster the product is installed to (for k8s clusters this should be `kube-system` ns UID)
	GetSystemUUID() string
	// GetProductUUID returns a product install UUID related to that specific installation of the product (e.g Ranchers per install unique `installuuid`).
	GetProductUUID() string
	GetProduct() products.OperatorProduct
	GetProductTripletValues() (string, string, string)
	MetricsToSystemInformation() registration.SystemInformation
}

type RancherMetricsWrapper struct {
	Product        products.OperatorProduct
	Data           map[string]any
	clusterUUID    string
	installUUID    string
	ProductVersion semver.Version
}

var _ MetricsWrapper = &RancherMetricsWrapper{}

func NewMetricsWrapper(data map[string]any) RancherMetricsWrapper {
	subscriptionData := data["subscription"].(map[string]interface{})

	rancherVersion := semver.Version(subscriptionData["version"].(string))
	return RancherMetricsWrapper{
		ProductVersion: rancherVersion,
		Product: products.OperatorProduct{
			Identifier: subscriptionData["product"].(string),
			Version:    rancherVersion.SCCSafeVersion(),
			Arch:       subscriptionData["arch"].(string),
		},
		Data:        data,
		clusterUUID: subscriptionData["clusteruuid"].(string),
		installUUID: subscriptionData["installuuid"].(string),
	}
}

func (mw *RancherMetricsWrapper) GetSystemUUID() string {
	return mw.clusterUUID
}

func (mw *RancherMetricsWrapper) GetProductUUID() string {
	return mw.installUUID
}

func (mw *RancherMetricsWrapper) GetProduct() products.OperatorProduct {
	return mw.Product
}

func (mw *RancherMetricsWrapper) GetProductTripletValues() (string, string, string) {
	return mw.Product.GetTripletValues()
}

func (mw *RancherMetricsWrapper) MetricsToSystemInformation() registration.SystemInformation {
	return mw.Data
}
