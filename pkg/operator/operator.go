package operator

import (
	"context"
	"fmt"

	corev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/ratelimit"
	"k8s.io/client-go/rest"

	"github.com/rancher/scc-operator/internal/initializer"
	"github.com/rancher/scc-operator/internal/log"
	"github.com/rancher/scc-operator/internal/types"
	"github.com/rancher/scc-operator/internal/wrangler"
	"github.com/rancher/scc-operator/pkg/crds"
	"github.com/rancher/scc-operator/pkg/generated/controllers/scc.cattle.io"
)

type SccOperator struct {
	devMode            bool
	log                log.StructuredLogger
	sccResourceFactory *scc.Factory
	secrets            corev1.SecretController
	options            *types.RunOptions
}

func New(
	ctx context.Context,
	kubeconfig *rest.Config,
	options types.RunOptions,
) (*SccStarter, error) {
	starterLog := options.Logger.WithField("component", "scc-starter")
	starterLog.Debug("Preparing to setup SCC operator")

	if err := options.Validate(); err != nil {
		return nil, err
	}
	initializer.OperatorName.Set(options.OperatorName)

	kubeconfig.RateLimiter = ratelimit.None
	wContext, err := wrangler.NewWranglerMiniContext(
		ctx,
		kubeconfig,
		options.SystemNamespace,
		options.LeaseNamespace,
	)
	if err != nil {
		return nil, err
	}

	starterLog.Debug("Setting up CRD Manager")
	crdManager := crds.NewCRDManager(
		options,
		wContext.ClientSet.ApiextensionsV1().CustomResourceDefinitions(),
	)

	ensureCrdErr := crdManager.EnsureRequired(ctx)
	if ensureCrdErr != nil {
		return nil, fmt.Errorf("failed to ensure required CRDs: %w", ensureCrdErr)
	}

	return &SccStarter{
		context:                 ctx,
		options:                 options,
		wrangler:                wContext,
		log:                     starterLog,
		systemRegistrationReady: make(chan struct{}),
	}, nil
}
