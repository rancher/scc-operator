package helpers

import (
	"github.com/rancher/wrangler/v3/pkg/generic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/scc-operator/internal/consts"
)

// These are helpers related to consts.LabelK8sManagedBy used to track resource ownership.

// SccManagedByValue constructs the SCC managed-by label value in the format "<operator>_secret-broker"
func SccManagedByValue(operatorName string) string {
	return operatorName + "_" + consts.ManagedByValueSecretBroker
}

func HasManagedByLabel[T metav1.Object](incomingObj T) bool {
	objectLabels := incomingObj.GetLabels()
	_, hasManagedBy := objectLabels[consts.LabelK8sManagedBy]

	return hasManagedBy
}

func GetManagedByValue[T metav1.Object](incomingObj T) string {
	objectLabels := incomingObj.GetLabels()
	return objectLabels[consts.LabelK8sManagedBy]
}

// ShouldManage will verify that this operator should manage a given object.
// Checks both k8s and SCC managed-by labels. Treats Helm as manageable.
func ShouldManage[T metav1.Object](incomingObj T, expectedManager string) bool {
	objectLabels := incomingObj.GetLabels()
	managedBy, hasManagedBy := objectLabels[consts.LabelK8sManagedBy]
	managedBySCC, hasManagedBySCC := objectLabels[consts.LabelSccManagedBy]
	expectedSCCManager := SccManagedByValue(expectedManager)

	// Has k8s label only (backwards compatibility)
	if hasManagedBy && !hasManagedBySCC {
		return managedBy == expectedManager
	}

	// Has both labels
	if hasManagedBy && hasManagedBySCC {
		return managedBySCC == expectedSCCManager && (managedBy == expectedManager || managedBy == "Helm")
	}

	return false
}

// TakeOwnership sets the k8s and SCC managed-by labels.
// Preserves app.kubernetes.io/managed-by if it's set to "Helm".
func TakeOwnership[T generic.RuntimeMetaObject](incomingObj T, owner string) T {
	objectLabels := incomingObj.GetLabels()
	if objectLabels == nil {
		objectLabels = map[string]string{
			consts.LabelK8sManagedBy: owner,
			consts.LabelSccManagedBy: SccManagedByValue(owner),
		}
	} else {
		// Only overwrite k8s managed-by if it's not Helm
		if objectLabels[consts.LabelK8sManagedBy] != "Helm" {
			objectLabels[consts.LabelK8sManagedBy] = owner
		}
		objectLabels[consts.LabelSccManagedBy] = SccManagedByValue(owner)
	}

	incomingObj.SetLabels(objectLabels)
	return incomingObj
}

// HasSccManagedByLabel checks if the SCC-specific managed-by label is set
func HasSccManagedByLabel[T metav1.Object](incomingObj T) bool {
	objectLabels := incomingObj.GetLabels()
	_, hasManagedBy := objectLabels[consts.LabelSccManagedBy]
	return hasManagedBy
}

// GetSccManagedByValue returns the value of scc.cattle.io/managed-by
func GetSccManagedByValue[T metav1.Object](incomingObj T) string {
	objectLabels := incomingObj.GetLabels()
	return objectLabels[consts.LabelSccManagedBy]
}
