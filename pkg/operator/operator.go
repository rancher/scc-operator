package operator

import (
	"context"
	"github.com/rancher-sandbox/scc-operator/internal/types"
	"github.com/rancher-sandbox/scc-operator/internal/util"
	rest "k8s.io/client-go/rest"
)

func Run(
	ctx context.Context,
	config *rest.Config,
	options types.RunOptions,
) error {
	util.SetSystemNamespace(options.SccNamespace)

	return nil
}
