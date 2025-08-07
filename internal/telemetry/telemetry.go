package telemetry

import (
	"context"
	"fmt"
	"reflect"

	"github.com/rancher/scc-operator/internal/consts"
	v1 "github.com/rancher/scc-operator/internal/rancher/apis/telemetry.cattle.io/v1"
	telemetryV1 "github.com/rancher/scc-operator/internal/rancher/generated/controllers/telemetry.cattle.io/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SecretRequester struct {
	secretRequests telemetryV1.SecretRequestController
	labels         map[string]string
}

func NewSecretRequester(secretRequests telemetryV1.SecretRequestController, labels map[string]string) *SecretRequester {
	return &SecretRequester{
		secretRequests: secretRequests,
		labels:         labels,
	}
}

func (s *SecretRequester) prepareSecretRequest() *v1.SecretRequest {
	secretRef := corev1.SecretReference{
		Namespace: consts.DefaultSCCNamespace,
		Name:      consts.SCCMetricsOutputSecretName,
	}
	return &v1.SecretRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:   consts.SCCMetricsOutputSecretName,
			Labels: s.labels,
		},
		Spec: v1.SecretRequestSpec{
			SecretType:      "scc",
			TargetSecretRef: &secretRef,
		},
	}
}

func (s *SecretRequester) EnsureSecretRequest(_ context.Context) error {
	desiredSecretRequest := s.prepareSecretRequest()
	logrus.Debug(desiredSecretRequest)

	existing, getErr := s.secretRequests.Get(consts.SCCMetricsOutputSecretName, metav1.GetOptions{})
	if getErr != nil && !errors.IsNotFound(getErr) {
		return fmt.Errorf("get secret request %s failed: %w", consts.SCCMetricsOutputSecretName, getErr)
	}

	if errors.IsNotFound(getErr) {
		_, err := s.secretRequests.Create(desiredSecretRequest)
		if err != nil {
			return fmt.Errorf("create secret request %s failed: %w", consts.SCCMetricsOutputSecretName, err)
		}

		return nil
	}

	if !reflect.DeepEqual(existing, desiredSecretRequest) {
		// TODO: update/patch existing
		return nil
	}

	// existing and desired match; noop
	return nil
}
