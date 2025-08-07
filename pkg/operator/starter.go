package operator

import (
	"context"
	"time"

	"github.com/rancher/scc-operator/internal/consts"
	"github.com/rancher/scc-operator/internal/rancher"
	"github.com/rancher/scc-operator/internal/telemetry"
	"github.com/rancher/wrangler/v3/pkg/start"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/rancher/scc-operator/internal/initializer"
	rootLog "github.com/rancher/scc-operator/internal/log"
	"github.com/rancher/scc-operator/internal/wrangler"
	"github.com/rancher/scc-operator/pkg/controllers"
)

type SccStarter struct {
	context                 context.Context
	wrangler                wrangler.MiniContext
	log                     rootLog.StructuredLogger
	systemRegistrationReady chan struct{}
}

func (s *SccStarter) CanStartSccOperator() bool {
	// TODO: add back the condition for local cluster?
	return s.isServerUrlReady() && s.hasSccMetricsSecretPopulated()
}

func (s *SccStarter) isServerUrlReady() bool {
	serverUrl := rancher.GetServerURL(s.context, s.wrangler.Settings)
	return serverUrl != ""
}

func (s *SccStarter) hasSccMetricsSecretPopulated() bool {
	return s.wrangler.Secrets.HasSecret(consts.DefaultSCCNamespace, consts.SCCMetricsOutputSecretName)
}

func (s *SccStarter) EnsureMetricsSecretRequest(ctx context.Context) error {
	labels := map[string]string{
		consts.LabelK8sManagedBy: consts.DefaultOperatorName,
	}
	metricsRequester := telemetry.NewSecretRequester(
		labels,
		s.wrangler.Dynamic,
	)
	return metricsRequester.EnsureSecretRequest(ctx)
}

func (s *SccStarter) waitForSystemReady(onSystemReady func()) {
	// Currently we only wait for ServerUrl not being empty, this is a good start as without the URL we cannot start.
	// However, we should also consider other state that we "need" to register with SCC like metrics about nodes/clusters.
	defer onSystemReady()
	if s.CanStartSccOperator() {
		close(s.systemRegistrationReady)
		return
	}

	s.log.Info("Waiting for server-url and/or initial metrics to be ready")
	wait.Until(func() {
		if s.CanStartSccOperator() {
			s.log.Info("can now start controllers; server URL and initial metrics are now ready.")
			close(s.systemRegistrationReady)
		} else {
			s.log.Info("cannot start controllers yet; server URL and/or initial metrics are not ready.")
		}
	}, 15*time.Second, s.systemRegistrationReady)
}

func (s *SccStarter) SetupControllers() error {
	go s.waitForSystemReady(func() {
		s.log.Debug("Setting up SCC Operator")
		initOperator, err := setup(s.context, &s.wrangler, s.log)
		if err != nil {
			s.log.Errorf("error setting up scc operator: %s", err.Error())
		}

		controllers.Register(
			s.context,
			initializer.OperatorName.Get(),
			initializer.SystemNamespace.Get(),
			initOperator.sccResourceFactory.Scc().V1().Registration(),
			s.wrangler.Secrets,
		)

		if startErr := start.All(s.context, 2, initOperator.sccResourceFactory); startErr != nil {
			s.log.Errorf("error starting operator: %v", startErr)
		}
		<-s.context.Done()
	})

	if s.systemRegistrationReady != nil {
		s.log.Info("SCC operator initialized; controllers waiting to start until system is ready")
	}

	return nil
}

func (s *SccStarter) Run() error {
	s.log.Debug("Starting to run SCC Operator; will only activate on leader")
	s.wrangler.OnLeader(func(_ context.Context) error {
		s.log.Debug("Preparing SCC controllers and starting them up")
		return s.SetupControllers()
	})

	return s.wrangler.Start(s.context)
}
