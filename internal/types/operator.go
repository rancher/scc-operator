package types

import (
	"fmt"

	rootLog "github.com/rancher/scc-operator/internal/log"
)

type OperatorMetadata struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildDate string `json:"buildDate"`
}

type RunOptions struct {
	Logger           rootLog.StructuredLogger
	OperatorName     string
	DevMode          bool
	OperatorMetadata OperatorMetadata
	SystemNamespace  string
	LeaseNamespace   string
}

func (o *RunOptions) Validate() error {
	if o.OperatorName == "" {
		return fmt.Errorf("operator name must be set")
	}
	// TODO: should we validate the NS exists? How should mgmt of this be handled?
	if o.SystemNamespace == "" {
		return fmt.Errorf("operator must have a valid SCC system namespace")
	}
	if o.LeaseNamespace == "" {
		o.Logger.Warn("operator lease namespace is empty; will default to `kube-system`")
	}
	return nil
}

type OperatorOptions interface{}
