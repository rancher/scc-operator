package helpers

import (
	"reflect"

	"github.com/rancher/wrangler/v3/pkg/generic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/scc-operator/internal/consts"
)

func HasManagedByLabel[T metav1.Object](incomingObj T) bool {
	objectAnnotations := incomingObj.GetAnnotations()
	_, hasManagedBy := objectAnnotations[consts.LabelSccManagedBy]

	return hasManagedBy
}

// ShouldManage will verify that this operator should manage a given object
func ShouldManage[T metav1.Object](incomingObj T, expectedManager string) bool {
	objectLabels := incomingObj.GetLabels()
	managedBy, hasManagedBy := objectLabels[consts.LabelK8sManagedBy]
	if !hasManagedBy {
		return false
	}

	if managedBy == expectedManager {
		return true
	}

	return false
}

// TakeOwnership will set or overwrite the value of the k8s managed-by label
func TakeOwnership[T generic.RuntimeMetaObject](incomingObj T, owner string) T {
	if isEmptyObject(incomingObj) {
		return incomingObj
	}

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

func isEmptyObject(obj interface{}) bool {
	if obj == nil {
		return true
	}

	objectType := reflect.TypeOf(obj).Elem()
	zeroValue := reflect.Zero(objectType).Interface()
	indirect := reflect.Indirect(reflect.ValueOf(obj)).Interface()
	return reflect.DeepEqual(indirect, zeroValue)
}
