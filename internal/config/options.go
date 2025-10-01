package config

import "github.com/rancher/scc-operator/internal/config/option"

var (
	Kubeconfig        = option.NewOption("kubeconfig", "")
	OperatorName      = option.NewOption("operator-name", "")
	OperatorNamespace = option.NewOption("operator-namespace", "")
	LeaseNamespace    = option.NewOption("lease-namespace", "")
	LogLevel          = option.NewOption("log-level", "", option.AllowedFromConfigMap)
	LogFormat         = option.NewOption("log-format", "", option.AllowedFromConfigMap)
	DevMode           = option.NewOption("dev-mode", false, option.AllowedFromConfigMap)
	RancherDevMode    = option.NewOption("rancher-dev-mode", false, option.AllowedFromConfigMap)
	Debug             = option.NewOption("debug", false, option.AllowedFromConfigMap)
	Trace             = option.NewOption("trace", false, option.AllowedFromConfigMap)

	// These are Dev Specific options that trigger SCC API to be staging by default
	ProductOverride        = option.NewOption("product-override", "")
	ProductVersionOverride = option.NewOption("product-version-override", "")
)
