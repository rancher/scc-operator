package operator

import (
	"context"
	"github.com/rancher/scc-operator/internal/log"
	"github.com/rancher/scc-operator/internal/telemetry"
	"github.com/rancher/scc-operator/internal/types"
	"github.com/rancher/scc-operator/internal/util"
	"github.com/rancher/scc-operator/internal/wrangler"
	"github.com/rancher/scc-operator/pkg/generated/controllers/scc.cattle.io"
	"github.com/rancher/scc-operator/pkg/systeminfo"
	corev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/ratelimit"
	rest "k8s.io/client-go/rest"
)

type SccOperator struct {
	devMode            bool
	log                log.StructuredLogger
	sccResourceFactory *scc.Factory
	secrets            corev1.SecretController
	rancherTelemetry   telemetry.TelemetryGatherer
}

func New(
	ctx context.Context,
	kubeconfig *rest.Config,
	options types.RunOptions,
) (*SccStarter, error) {
	operatorLogger := options.Logger
	operatorLogger.Debug("Preparing to setup SCC operator")

	util.SetSystemNamespace(options.SccNamespace)
	if err := options.Validate(); err != nil {
		return nil, err
	}

	kubeconfig.RateLimiter = ratelimit.None
	wContext, err := wrangler.NewWranglerMiniContext(ctx, kubeconfig)
	if err != nil {
		return nil, err
	}

	infoProvider := systeminfo.NewInfoProvider(wContext.Settings, wContext.Mgmt.Node().Cache())

	return &SccStarter{
		context:                 ctx,
		wrangler:                wContext,
		log:                     operatorLogger.WithField("component", "scc-starter"),
		systemInfoProvider:      infoProvider,
		systemRegistrationReady: make(chan struct{}),
	}, nil
}
