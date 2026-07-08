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

// ShouldManage will verify that this operator should manage a given object
func ShouldManage[T metav1.Object](incomingObj T, expectedManager string) bool {
	objectLabels := incomingObj.GetLabels()
	managedBy, hasManagedBy := objectLabels[consts.LabelK8sManagedBy]

	return hasManagedBy && managedBy == expectedManager
}

// TakeOwnership will set or overwrite the value of the k8s managed-by label
func TakeOwnership[T generic.RuntimeMetaObject](incomingObj T, owner string) T {
	objectLabels := incomingObj.GetLabels()
	if objectLabels == nil {
		objectLabels = map[string]string{
			consts.LabelK8sManagedBy: owner,
		}
	} else {
		objectLabels[consts.LabelK8sManagedBy] = owner
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

// ShouldManageByScc checks if this operator should manage based on SCC label.
// Falls back to k8s managed-by for backwards compatibility.
// Special case: Helm-managed resources are treated as manageable by this operator.
func ShouldManageByScc[T metav1.Object](incomingObj T, expectedManager string) bool {
	objectLabels := incomingObj.GetLabels()

	// If the caller passes the SCC managed-by value (e.g. "<operator>_secret-broker"),
	// derive the expected k8s managed-by value ("<operator>") for fallback behavior.
	expectedK8sManager := expectedManager
	suffix := "_" + consts.ManagedByValueSecretBroker
	if len(expectedManager) > len(suffix) && expectedManager[len(expectedManager)-len(suffix):] == suffix {
		expectedK8sManager = expectedManager[:len(expectedManager)-len(suffix)]
	}

	// Check SCC-specific label first (new behavior)
	sccManagedBy, hasSccManagedBy := objectLabels[consts.LabelSccManagedBy]
	if hasSccManagedBy {
		return sccManagedBy == expectedManager
	}

	// Check k8s managed-by
	k8sManagedBy, hasK8sManagedBy := objectLabels[consts.LabelK8sManagedBy]
	if hasK8sManagedBy {
		// Treat Helm-managed resources as manageable by this operator.
		// This allows Helm-deployed entrypoint secrets to be processed without requiring
		// the operator to overwrite app.kubernetes.io/managed-by.
		if k8sManagedBy == "Helm" {
			return true
		}
		// Fall back to exact match for backwards compatibility
		return k8sManagedBy == expectedK8sManager
	}

	// If neither label is set, resource is unmanaged
	return false
}

// TakeSccOwnership sets the SCC managed-by label without touching k8s managed-by.
// Use this to adopt resources that may be managed by other tools (e.g., Helm).
func TakeSccOwnership[T generic.RuntimeMetaObject](incomingObj T, owner string) T {
	objectLabels := incomingObj.GetLabels()
	if objectLabels == nil {
		objectLabels = map[string]string{
			consts.LabelSccManagedBy: owner,
		}
	} else {
		objectLabels[consts.LabelSccManagedBy] = owner
	}

	incomingObj.SetLabels(objectLabels)
	return incomingObj
}

// TakeFullOwnership sets both SCC and k8s managed-by labels.
// Use this for resources created and fully owned by the operator.
func TakeFullOwnership[T generic.RuntimeMetaObject](incomingObj T, owner string) T {
	objectLabels := incomingObj.GetLabels()
	if objectLabels == nil {
		objectLabels = map[string]string{}
	}

	objectLabels[consts.LabelSccManagedBy] = owner
	objectLabels[consts.LabelK8sManagedBy] = owner

	incomingObj.SetLabels(objectLabels)
	return incomingObj
}
