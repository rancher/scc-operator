package operator

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	k8sv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/rancher/scc-operator/internal/log"
	"github.com/rancher/scc-operator/internal/repos/settingrepo"
	"github.com/rancher/scc-operator/internal/telemetry"
	"github.com/rancher/scc-operator/internal/util"
	"github.com/rancher/scc-operator/internal/wrangler"
	"github.com/rancher/scc-operator/pkg/generated/controllers/scc.cattle.io"
	"github.com/rancher/scc-operator/pkg/systeminfo"
)

func setup(wContext *wrangler.MiniContext, logger log.StructuredLogger, infoProvider *systeminfo.InfoProvider) (*SccOperator, error) {
	namespaces := wContext.Core.Namespace()
	var kubeSystemNS *k8sv1.Namespace

	kubeNsErr := retry.OnError(
		retry.DefaultRetry,
		func(err error) bool {
			return apierrors.IsNotFound(err)
		},
		func() error {
			maybeNs, err := namespaces.Get("kube-system", metav1.GetOptions{})
			if err != nil {
				return err
			}

			kubeSystemNS = maybeNs
			return nil
		},
	)

	if kubeNsErr != nil {
		return nil, fmt.Errorf("failed to get kube-system namespace: %v", kubeNsErr)
	}

	rancherUUID := settingrepo.GetRancherInstallUUID(wContext.Settings)
	if rancherUUID == "" {
		err := errors.New("no rancher uuid found")
		logger.Fatalf("Error getting rancher uuid: %v", err)
		return nil, err
	}

	sccResources, err := scc.NewFactoryFromConfig(wContext.RESTConfig)
	if err != nil {
		logger.Fatalf("Error getting scc resources: %v", err)
		return nil, err
	}
	// Validate that the UUID is in the correct format
	parsedRancherUUID, rancherUUIDErr := uuid.Parse(rancherUUID)
	parsedkubeSystemNSUID, kubeUUIDErr := uuid.Parse(string(kubeSystemNS.UID))

	if rancherUUIDErr != nil || kubeUUIDErr != nil {
		return nil, fmt.Errorf("invalid UUID format: rancherUUID=%s, kubeSystemNS.UID=%s", rancherUUID, string(kubeSystemNS.UID))
	}
	infoProvider = infoProvider.SetUuids(parsedRancherUUID, parsedkubeSystemNSUID)

	rancherTelemetry := telemetry.NewTelemetryGatherer(wContext)

	return &SccOperator{
		devMode:            util.DevMode.Get(),
		log:                logger,
		sccResourceFactory: sccResources,
		secrets:            wContext.Core.Secret(),
		rancherTelemetry:   rancherTelemetry,
	}, nil
}
