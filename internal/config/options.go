package config

import (
	"github.com/rancher/scc-operator/internal/config/option"
	"github.com/rancher/scc-operator/internal/consts"
)

var (
	Kubeconfig        = option.NewOption("kubeconfig", "")
	OperatorName      = option.NewOption("operator-name", consts.DefaultOperatorName)
	OperatorNamespace = option.NewOption("operator-namespace", consts.DefaultSCCNamespace)
	LeaseNamespace    = option.NewOption("lease-namespace", consts.DefaultLeaseNamespace)
	LogLevel          = option.NewOption("log-level", "", option.AllowedFromConfigMap)
	LogFormat         = option.NewOption("log-format", "", option.AllowedFromConfigMap)
	DevMode           = option.NewOption("dev-mode", false, option.AllowedFromConfigMap)
	RancherDevMode    = option.NewOption("rancher-dev-mode", false, option.AllowedFromConfigMap)
	Debug             = option.NewOption("debug", false, option.AllowedFromConfigMap)
	Trace             = option.NewOption("trace", false, option.AllowedFromConfigMap)
)
