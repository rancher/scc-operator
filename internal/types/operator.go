package types

import (
	"github.com/rancher/scc-operator/internal/config"
	rootLog "github.com/rancher/scc-operator/internal/logging"
)

// OperatorMetadata contains the basic build info about the operator binary version info
type OperatorMetadata struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildDate string `json:"buildDate"`
}

// RunOptions holds all the necessary CLI run options and configs
type RunOptions struct {
	Logger           rootLog.StructuredLogger
	OperatorName     string // OperatorName is intentionally redundant and set by OperatorSettings
	OperatorSettings *config.OperatorSettings
	DevMode          bool
	OperatorMetadata OperatorMetadata
}

func (o *RunOptions) Validate() error {
	if err := o.OperatorSettings.Validate(); err != nil {
		return err
	}

	return nil
}

func (o *RunOptions) SystemNamespace() string {
	return o.OperatorSettings.SystemNamespace
}
