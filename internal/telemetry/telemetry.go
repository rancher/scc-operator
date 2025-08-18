package telemetry

import (
	"context"
	"fmt"
	"reflect"

	"github.com/rancher/scc-operator/internal/consts"
	"github.com/rancher/scc-operator/internal/initializer"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type SecretRequester struct {
	labels                     map[string]string
	secretRequestDynamicClient dynamic.NamespaceableResourceInterface
}

func NewSecretRequester(
	labels map[string]string,
	dynamicClient dynamic.Interface,
) *SecretRequester {
	return &SecretRequester{
		labels:                     labels,
		secretRequestDynamicClient: dynamicClient.Resource(telemetrySecretRequestGVR()),
	}
}

const (
	RancherTelemetryGroup                 = "telemetry.cattle.io"
	RancherTelemetryVersion               = "v1"
	RancherTelemetrySecretRequestResource = "secretrequests"
)

func telemetrySecretRequestGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    RancherTelemetryGroup,
		Version:  RancherTelemetryVersion,
		Resource: RancherTelemetrySecretRequestResource,
	}
}

func (s *SecretRequester) prepareSecretRequestUnstructured() *unstructured.Unstructured {
	gvr := telemetrySecretRequestGVR()
	desiredSecretRequest := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": gvr.GroupVersion().Identifier(),
			"kind":       "SecretRequest",
			"metadata": map[string]interface{}{
				"name": consts.RancherMetricsSecretRequestName,
			},
			"spec": map[string]interface{}{
				"secretType": "scc",
				"targetSecretRef": map[string]interface{}{
					"namespace": initializer.SystemNamespace.Get(),
					"name":      consts.SCCMetricsOutputSecretName,
				},
			},
		},
	}
	desiredSecretRequest.SetFinalizers([]string{
		consts.FinalizerSccMetricsSecretRequest,
	})
	desiredSecretRequest.SetLabels(s.labels)

	return &desiredSecretRequest
}

func (s *SecretRequester) EnsureSecretRequest(ctx context.Context) error {
	desiredSecretRequest := s.prepareSecretRequestUnstructured()
	logrus.Debug(desiredSecretRequest)

	existing, getErr := s.secretRequestDynamicClient.Get(ctx, desiredSecretRequest.GetName(), metav1.GetOptions{})
	if getErr != nil && !errors.IsNotFound(getErr) {
		return fmt.Errorf("get secret request %s failed: %w", consts.RancherMetricsSecretRequestName, getErr)
	}

	if errors.IsNotFound(getErr) {
		_, err := s.secretRequestDynamicClient.Create(ctx, desiredSecretRequest, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create secret request %s failed: %w", consts.RancherMetricsSecretRequestName, err)
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
