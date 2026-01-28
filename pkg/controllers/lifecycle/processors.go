package lifecycle

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/scc-operator/internal/types"
	v1 "github.com/rancher/scc-operator/pkg/apis/scc.cattle.io/v1"
)

var (
	registrationProcessors       []types.RegistrationProcessor
	registrationStatusProcessors = []types.RegistrationStatusProcessor{
		PrepareSuccessfulActivation,
	}
)

func PrepareSuccessfulActivation(regIn *v1.Registration) *v1.Registration {
	now := metav1.Now()
	v1.RegistrationConditionActivated.True(regIn)
	v1.ResourceConditionProgressing.False(regIn)
	v1.ResourceConditionReady.True(regIn)
	v1.ResourceConditionDone.True(regIn)
	regIn.Status.ActivationStatus.LastValidatedTS = &now
	regIn.Status.ActivationStatus.Activated = true

	// Set ResourceConditionDone as the CurrentCondition since it represents the final successful state
	regIn.SetCurrentCondition(v1.ResourceConditionDone)

	return regIn
}
