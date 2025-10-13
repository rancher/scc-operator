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
	managedBy, hasManagedBy := objectLabels[consts.LabelK8sManagedBy]
	if !hasManagedBy {
		return ""
	}

	return managedBy
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
