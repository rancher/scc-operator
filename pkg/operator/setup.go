package operator

import (
	"fmt"

	"github.com/google/uuid"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/rancher-sandbox/scc-operator/internal/log"
	"github.com/rancher-sandbox/scc-operator/internal/util"
	"github.com/rancher-sandbox/scc-operator/pkg/generated/controllers/scc.cattle.io"
	"github.com/rancher-sandbox/scc-operator/pkg/systeminfo"
	corev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
)

type sccOperator struct {
	devMode            bool
	log                log.StructuredLogger
	sccResourceFactory *scc.Factory
	secrets            corev1.SecretController
	rancherTelemetry   telemetry.TelemetryGatherer
}

func setup(wContext *wrangler.Context, logger log.StructuredLogger, infoProvider *systeminfo.InfoProvider) (*sccOperator, error) {
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

	rancherUuid := settings.InstallUUID.Get()
	if rancherUuid == "" {
		err := errors.New("no rancher uuid found")
		logger.Fatalf("Error getting rancher uuid: %v", err)
		return nil, err
	}

	sccResources, err := scc.NewFactoryFromConfig(wContext.RESTConfig)
	if err != nil {
		logger.Fatalf("Error getting scc resources: %v", err)
		return nil, err
	}
	// Validate that the UUID is in correct format
	parsedRancherUUID, rancherUuidErr := uuid.Parse(rancherUuid)
	parsedkubeSystemNSUID, kubeUuidErr := uuid.Parse(string(kubeSystemNS.UID))

	if rancherUuidErr != nil || kubeUuidErr != nil {
		return nil, fmt.Errorf("invalid UUID format: rancherUuid=%s, kubeSystemNS.UID=%s", rancherUuid, string(kubeSystemNS.UID))
	}
	infoProvider = infoProvider.SetUuids(parsedRancherUUID, parsedkubeSystemNSUID)

	rancherVersion := systeminfo.GetVersion()
	rancherTelemetry := telemetry.NewTelemetryGatherer(rancherVersion, wContext.Mgmt.Cluster().Cache(), wContext.Mgmt.Node().Cache())

	return &sccOperator{
		devMode:            util.DevMode(),
		log:                logger,
		sccResourceFactory: sccResources,
		secrets:            wContext.Core.Secret(),
		rancherTelemetry:   rancherTelemetry,
	}, nil
}
