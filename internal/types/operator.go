package types

type RunOptions struct {
	SccNamespace string
}

func (o *RunOptions) Validate() error {
	// TODO: any necessary validation worth throwing errors on
	return nil
}

type OperatorOptions interface{}
