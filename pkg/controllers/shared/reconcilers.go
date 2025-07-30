package shared

import (
	"github.com/rancher/scc-operator/internal/types"
	v1 "github.com/rancher/scc-operator/pkg/apis/scc.cattle.io/v1"
)

var (
	registrationReconciler = []types.RegistrationFailureReconciler{
		PrepareFailed,
	}
)

func PrepareFailed(regIn *v1.Registration, err error) *v1.Registration {
	v1.ResourceConditionProgressing.False(regIn)
	v1.ResourceConditionReady.False(regIn)
	v1.ResourceConditionFailure.True(regIn)
	v1.ResourceConditionFailure.SetError(regIn, "could not complete registration", err)

	return regIn
}
