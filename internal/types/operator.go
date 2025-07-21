package types

import rootLog "github.com/rancher/scc-operator/internal/log"

type RunOptions struct {
	Logger       rootLog.StructuredLogger
	SccNamespace string
}

func (o *RunOptions) Validate() error {
	// TODO: any necessary validation worth throwing errors on
	return nil
}

type OperatorOptions interface{}
