package helpers

import (
	"github.com/rancher/wrangler/v3/pkg/generic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/scc-operator/internal/consts"
)

// These are helpers related to consts.LabelK8sManagedBy used to track resource ownership.

func HasManagedByLabel[T metav1.Object](incomingObj T) bool {
	objectLabels := incomingObj.GetLabels()
	_, hasManagedBy := objectLabels[consts.LabelK8sManagedBy]

	return hasManagedBy
}

func GetManagedByValue[T metav1.Object](incomingObj T) string {
	objectLabels := incomingObj.GetLabels()
	return objectLabels[consts.LabelK8sManagedBy]
}

// ShouldAdopt checks if this operator should adopt/take ownership of a resource.
// Returns true for unmanaged resources, Helm-managed resources, or resources already managed by this operator.
func ShouldAdopt[T metav1.Object](incomingObj T, expectedManager string) bool {
	objectLabels := incomingObj.GetLabels()
	managedBy, hasManagedBy := objectLabels[consts.LabelK8sManagedBy]
	managedBySCC, hasManagedBySCC := objectLabels[consts.LabelSccManagedBy]
	expectedSCCManager := consts.SccManagedByValue(expectedManager)

	// No labels at all - unmanaged resource, should adopt
	if !hasManagedBy && !hasManagedBySCC {
		return true
	}

	// Has k8s label only (backwards compatibility)
	if hasManagedBy && !hasManagedBySCC {
		return managedBy == expectedManager || managedBy == "Helm"
	}

	// Has SCC label only (manual edit or corruption - trust our label)
	if !hasManagedBy && hasManagedBySCC {
		return managedBySCC == expectedSCCManager
	}

	// Has both labels
	if hasManagedBy && hasManagedBySCC {
		return managedBySCC == expectedSCCManager && (managedBy == expectedManager || managedBy == "Helm")
	}

	return false
}

// ShouldManage checks if this operator already manages a resource.
// Returns true only for resources already managed by this operator (not Helm-only).
func ShouldManage[T metav1.Object](incomingObj T, expectedManager string) bool {
	objectLabels := incomingObj.GetLabels()
	managedBy, hasManagedBy := objectLabels[consts.LabelK8sManagedBy]
	managedBySCC, hasManagedBySCC := objectLabels[consts.LabelSccManagedBy]
	expectedSCCManager := consts.SccManagedByValue(expectedManager)

	// Has k8s label only (backwards compatibility)
	if hasManagedBy && !hasManagedBySCC {
		return managedBy == expectedManager
	}

	// Has SCC label only (manual edit or corruption - trust our label)
	if !hasManagedBy && hasManagedBySCC {
		return managedBySCC == expectedSCCManager
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
			consts.LabelSccManagedBy: consts.SccManagedByValue(owner),
		}
	} else {
		// Only overwrite k8s managed-by if it's not Helm
		if objectLabels[consts.LabelK8sManagedBy] != "Helm" {
			objectLabels[consts.LabelK8sManagedBy] = owner
		}
		objectLabels[consts.LabelSccManagedBy] = consts.SccManagedByValue(owner)
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
