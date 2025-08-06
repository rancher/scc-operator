package types

import (
	"fmt"

	rootLog "github.com/rancher/scc-operator/internal/log"
)

type RunOptions struct {
	Logger         rootLog.StructuredLogger
	OperatorName   string
	SccNamespace   string
	LeaseNamespace string
}

func (o *RunOptions) Validate() error {
	if o.OperatorName == "" {
		return fmt.Errorf("operator name must be set")
	}
	// TODO: should we validate the NS exists? How should mgmt of this be handled?
	if o.SccNamespace == "" {
		return fmt.Errorf("operator must have a valid SCC namespace")
	}
	if o.LeaseNamespace == "" {
		o.Logger.Warn("operator lease namespace is empty; will default to `kube-system`")
	}
	return nil
}

type OperatorOptions interface{}
