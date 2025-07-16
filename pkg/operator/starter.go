package operator

import (
	rootLog "github.com/rancher-sandbox/scc-operator/internal/log"
	"github.com/rancher-sandbox/scc-operator/pkg/systeminfo"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

type sccStarter struct {
	log                     rootLog.StructuredLogger
	systemInfoProvider      *systeminfo.InfoProvider
	systemRegistrationReady chan struct{}
}

// TODO: in a standalone container we need to consider leadership/lease tracking
// Only one container of this type should hold the leadership role and actually start fully.
// Any non-leaders should be ready to start fully if they are promoted.
func (s *sccStarter) waitForSystemReady(onSystemReady func()) {
	// Currently we only wait for ServerUrl not being empty, this is a good start as without the URL we cannot start.
	// However, we should also consider other state that we "need" to register with SCC like metrics about nodes/clusters.
	defer onSystemReady()
	if s.systemInfoProvider != nil && s.systemInfoProvider.CanStartSccOperator() {
		close(s.systemRegistrationReady)
		return
	}
	s.log.Info("Waiting for server-url and/or local cluster to be ready")
	wait.Until(func() {
E		if s.systemInfoProvider != nil && s.systemInfoProvider.CanStartSccOperator() {
			s.log.Info("can now start controllers; server URL and local cluster are now ready.")
			close(s.systemRegistrationReady)
		} else {
			s.log.Info("cannot start controllers yet; server URL and/or local cluster are not ready.")
		}
	}, 15*time.Second, s.systemRegistrationReady)
}
