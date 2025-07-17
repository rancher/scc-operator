package common

import (
	"github.com/rancher-sandbox/scc-operator/internal/types"
	"github.com/rancher-sandbox/scc-operator/pkg/apis/scc.cattle.io/v1"
)

// GetRegistrationReconcilers returns all shared reconcilers
func GetRegistrationReconcilers() []types.RegistrationFailureReconciler {
	return []types.RegistrationFailureReconciler{
		PrepareFailed,
	}
}

func PrepareFailed(regIn *v1.Registration, err error) *v1.Registration {
	v1.ResourceConditionProgressing.False(regIn)
	v1.ResourceConditionReady.False(regIn)
	v1.ResourceConditionFailure.True(regIn)
	v1.ResourceConditionFailure.SetError(regIn, "could not complete registration", err)

	return regIn
}
