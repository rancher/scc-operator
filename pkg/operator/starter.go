package operator

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rancher/scc-operator/internal/consts"
	"github.com/rancher/scc-operator/internal/rancher"
	"github.com/rancher/scc-operator/internal/telemetry"
	"github.com/rancher/scc-operator/internal/types"
	"github.com/rancher/wrangler/v3/pkg/start"
	"k8s.io/apimachinery/pkg/util/wait"

	rootLog "github.com/rancher/scc-operator/internal/log"
	"github.com/rancher/scc-operator/internal/wrangler"
	"github.com/rancher/scc-operator/pkg/controllers"
)

// TODO(rancher-bias): all of the SCC starter/setup needs to not depend on product specific logic
type SccStarter struct {
	context                 context.Context
	wrangler                wrangler.MiniContext
	log                     rootLog.StructuredLogger
	systemRegistrationReady chan struct{}
	options                 types.RunOptions
}

func (s *SccStarter) CanStartSccOperator() bool {
	return s.isServerURLReady() && s.hasSccMetricsSecretPopulated()
}

// TODO(rancher-bias): Will other SCC Operator consumers have a Server URL?
func (s *SccStarter) isServerURLReady() bool {
	serverURL := rancher.GetServerURL(s.context, s.wrangler.Settings)
	if serverURL == "" {
		s.log.Trace("Server URL is not ready yet.")
		return false
	}
	s.log.Tracef("Server URL is ready: %s", serverURL)
	return true
}

// TODO(rancher-bias): Metrics Secret (for now) is just a Rancher thing - but maybe we should make it product universal?
func (s *SccStarter) hasSccMetricsSecretPopulated() bool {
	if !s.wrangler.Secrets.HasMetricsSecret() {
		s.log.Trace("Metrics secret is not populated yet.")
		return false
	}
	s.log.Trace("Metrics secret is populated.")
	return true
}

func (s *SccStarter) EnsureMetricsSecretRequest(ctx context.Context, namespace string) error {
	labels := map[string]string{
		consts.LabelK8sManagedBy: s.options.OperatorName,
	}
	metricsRequester := telemetry.NewSecretRequester(
		namespace,
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
		s.log.Debug("System is ready, closing systemRegistrationReady channel.")
		return
	}

	s.log.Info("Waiting for server-url and/or initial metrics to be ready")
	wait.Until(func() {
		if s.CanStartSccOperator() {
			s.log.Info("can now start controllers; server URL and initial metrics are now ready.")
			close(s.systemRegistrationReady)
		} else {
			s.log.Trace("cannot start controllers yet; checking readiness conditions...")
			s.isServerURLReady()             // This will log trace if not ready
			s.hasSccMetricsSecretPopulated() // This will log trace if not ready
		}
	}, 15*time.Second, s.systemRegistrationReady)
}

func (s *SccStarter) SetupControllers() error {
	// TODO(rancher-bias): The controller should start when the operator believes it is stable
	// Product specific bias must be applied only to specific Registration processing
	go s.waitForSystemReady(func() {
		s.log.Debug("Setting up SCC Operator")
		// TODO: remove rancher bias from operator startup
		initOperator, err := setup(s.context, s.options.Logger, &s.options, &s.wrangler)
		if err != nil {
			s.log.Errorf("error setting up scc operator: %s", err.Error())
		}

		// TODO: this can be split up by secrets and registrations - allowing secrets to register first
		// Registration controller should still wait until the metrics secret is available to start
		controllers.Register(
			s.context,
			&s.options,
			initOperator.sccResourceFactory.Scc().V1().Registration(),
			s.wrangler.Secrets,
			s.wrangler.Settings,
		)

		if startErr := start.All(s.context, consts.OperatorWorkerThreads, initOperator.sccResourceFactory); startErr != nil {
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
	s.log.Tracef("Attempting to acquire lease in namespace: %s", s.options.OperatorSettings.LeaseNamespace)
	s.wrangler.OnLeader(func(_ context.Context) error {
		s.log.Debug("Lease acquired. Preparing SCC controllers and starting them up")
		return s.SetupControllers()
	})

	return s.wrangler.Start(s.context)
}

func (s *SccStarter) StartMetricsAndHealthEndpoint() {
	// TODO(rancher-bias): this shouldn't be dependant on Rancher logic
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		// TODO: utilize more complex logic for ready condition & expose more info?
		if s.systemRegistrationReady != nil {
			w.WriteHeader(500)
			w.Write([]byte(fmt.Sprintf("error: %v", "some err here")))
		} else {
			w.WriteHeader(200)
			w.Write([]byte("ok"))
		}
	})

	http.ListenAndServe(":8080", nil)
}
