package config

import "github.com/rancher/scc-operator/internal/config/option"

var (
	Kubeconfig             = option.NewOption("kubeconfig", "")
	OperatorName           = option.NewOption("operator-name", "")
	OperatorNamespace      = option.NewOption("operator-namespace", "")
	LeaseNamespace         = option.NewOption("lease-namespace", "")
	LogLevel               = option.NewOption("log-level", "", option.AllowedFromConfigMap)
	LogFormat              = option.NewOption("log-format", "", option.AllowedFromConfigMap)
	RancherDevMode         = option.NewOption("rancher-dev-mode", false, option.AllowedFromConfigMap)
	Debug                  = option.NewOption("debug", false, option.AllowedFromConfigMap)
	Trace                  = option.NewOption("trace", false, option.AllowedFromConfigMap)
	Product                = option.NewOption("product", "")
	ProductVersion         = option.NewOption("product-version", "")
	ProductOverride        = option.NewOption("product-override", "")
	ProductVersionOverride = option.NewOption("product-version-override", "")
)
