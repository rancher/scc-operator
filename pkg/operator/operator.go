package operator

import (
	"context"
	"github.com/rancher-sandbox/scc-operator/internal/consts"
	"github.com/rancher-sandbox/scc-operator/internal/types"
	"github.com/rancher-sandbox/scc-operator/internal/util"
	"github.com/rancher-sandbox/scc-operator/pkg/systeminfo"
	"github.com/rancher/wrangler/v3/pkg/start"
	rest "k8s.io/client-go/rest"
	"k8s.io/component-base/metrics/prometheus/controllers"
)

func Run(
	ctx context.Context,
	config *rest.Config,
	options types.RunOptions,
) error {
	util.SetSystemNamespace(options.SccNamespace)

	operatorLogger := options.Logger
	operatorLogger.Debug("Preparing to setup SCC operator")

	infoProvider := systeminfo.NewInfoProvider(
		wContext.Mgmt.Node().Cache(),
	)

	starter := sccStarter{
		log:                     operatorLogger.WithField("component", "scc-starter"),
		systemInfoProvider:      infoProvider,
		systemRegistrationReady: make(chan struct{}),
	}

	go starter.waitForSystemReady(func() {
		operatorLogger.Debug("Setting up SCC Operator")
		initOperator, err := setup(wContext, operatorLogger, infoProvider)
		if err != nil {
			starter.log.Errorf("error setting up scc operator: %s", err.Error())
		}

		controllers.Register(
			ctx,
			consts.DefaultSCCNamespace,
			initOperator.sccResourceFactory.Scc().V1().Registration(),
			initOperator.secrets,
			initOperator.rancherTelemetry,
			infoProvider,
		)

		if err := start.All(ctx, 2, initOperator.sccResourceFactory); err != nil {
			initOperator.log.Errorf("error starting operator: %s", err.Error())
		}
	})

	if starter.systemRegistrationReady != nil {
		operatorLogger.Info("SCC operator initialized; controllers waiting to start until system is ready")
	}

	return nil
}
