package operator

import (
	"context"
	"github.com/rancher-sandbox/scc-operator/internal/types"
	"github.com/rancher-sandbox/scc-operator/internal/util"
	"github.com/rancher-sandbox/scc-operator/internal/wrangler"
	"github.com/rancher/wrangler/v3/pkg/ratelimit"
	"github.com/rancher/wrangler/v3/pkg/start"
	rest "k8s.io/client-go/rest"
)

func Run(
	ctx context.Context,
	kubeconfig *rest.Config,
	options types.RunOptions,
) error {
	operatorLogger := options.Logger
	operatorLogger.Debug("Preparing to setup SCC operator")

	util.SetSystemNamespace(options.SccNamespace)
	if err := options.Validate(); err != nil {
		return err
	}

	kubeconfig.RateLimiter = ratelimit.None
	wContext, err := wrangler.NewWranglerMiniContext(kubeconfig)
	if err != nil {
		return err
	}

	// TODO: init info provider
	// infoProvider := systeminfo.NewInfoProvider(
	// 	wContext.Mgmt.Node().Cache(),
	// )

	starter := sccStarter{
		log:                     operatorLogger.WithField("component", "scc-starter"),
		systemInfoProvider:      nil,
		systemRegistrationReady: make(chan struct{}),
	}

	go starter.waitForSystemReady(func() {
		operatorLogger.Debug("Setting up SCC Operator")
		initOperator, err := setup(&wContext, operatorLogger, nil)
		if err != nil {
			starter.log.Errorf("error setting up scc operator: %s", err.Error())
		}

		operatorLogger.Info("THIS IS WHERE I REGISTER CONTROLLERS")
		/*
			TODO: add operator code
			controllers.Register(
				ctx,
				consts.DefaultSCCNamespace,
				initOperator.sccResourceFactory.Scc().V1().Registration(),
				initOperator.secrets,
				initOperator.rancherTelemetry,
				infoProvider,
			)
		*/

		if err := start.All(ctx, 2, initOperator.sccResourceFactory); err != nil {
			operatorLogger.Errorf("error starting operator: %s", err.Error())
		}
		<-ctx.Done()
	})

	if starter.systemRegistrationReady != nil {
		operatorLogger.Info("SCC operator initialized; controllers waiting to start until system is ready")
	}

	return nil
}
