package operator

import (
	"context"
	rootLog "github.com/rancher/scc-operator/internal/log"
	"github.com/rancher/scc-operator/internal/util"
	"github.com/rancher/scc-operator/internal/wrangler"
	"github.com/rancher/scc-operator/pkg/controllers"
	"github.com/rancher/scc-operator/pkg/systeminfo"
	"github.com/rancher/wrangler/v3/pkg/start"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

type SccStarter struct {
	context                 context.Context
	wrangler                wrangler.MiniContext
	log                     rootLog.StructuredLogger
	systemInfoProvider      *systeminfo.InfoProvider
	systemRegistrationReady chan struct{}
}

// TODO: in a standalone container we need to consider leadership/lease tracking
// Only one container of this type should hold the leadership role and actually start fully.
// Any non-leaders should be ready to start fully if they are promoted.
func (s *SccStarter) waitForSystemReady(onSystemReady func()) {
	// Currently we only wait for ServerUrl not being empty, this is a good start as without the URL we cannot start.
	// However, we should also consider other state that we "need" to register with SCC like metrics about nodes/clusters.
	defer onSystemReady()
	if s.systemInfoProvider != nil && s.systemInfoProvider.CanStartSccOperator() {
		close(s.systemRegistrationReady)
		return
	}
	s.log.Info("Waiting for server-url and/or local cluster to be ready")
	wait.Until(func() {
		if s.systemInfoProvider != nil && s.systemInfoProvider.CanStartSccOperator() {
			s.log.Info("can now start controllers; server URL and local cluster are now ready.")
			close(s.systemRegistrationReady)
		} else {
			s.log.Info("cannot start controllers yet; server URL and/or local cluster are not ready.")
		}
	}, 15*time.Second, s.systemRegistrationReady)
}

func (s *SccStarter) SetupControllers() error {
	go s.waitForSystemReady(func() {
		s.log.Debug("Setting up SCC Operator")
		initOperator, err := setup(&s.wrangler, s.log, s.systemInfoProvider)
		if err != nil {
			s.log.Errorf("error setting up scc operator: %s", err.Error())
		}

		controllers.Register(
			s.context,
			util.SystemNamespace.Get(),
			initOperator.sccResourceFactory.Scc().V1().Registration(),
			s.wrangler.Secrets,
			initOperator.rancherTelemetry,
			s.systemInfoProvider,
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
	s.wrangler.OnLeader(func(ctx context.Context) error {
		s.log.Debug("[rancher::start] starting RancherSCCRegistrationExtension")
		return s.SetupControllers()
	})

	return s.wrangler.Start(s.context)
}
