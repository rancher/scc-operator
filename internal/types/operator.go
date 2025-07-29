package types

import (
	"fmt"
	rootLog "github.com/rancher/scc-operator/internal/log"
)

type RunOptions struct {
	Logger       rootLog.StructuredLogger
	OperatorName string
	SccNamespace string
}

func (o *RunOptions) Validate() error {
	// TODO: any necessary validation worth throwing errors on
	if o.OperatorName == "" {
		return fmt.Errorf("operator name must be set")
	}
	if o.SccNamespace == "" {
		return fmt.Errorf("operator must have a valid SCC namespace")
	}
	return nil
}

type OperatorOptions interface{}
