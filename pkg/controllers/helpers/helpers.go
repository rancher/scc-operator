package helpers

import (
	"reflect"
)

func isEmptyObject(obj interface{}) bool {
	if obj == nil {
		return true
	}

	objectType := reflect.TypeOf(obj).Elem()
	zeroValue := reflect.Zero(objectType).Interface()
	indirect := reflect.Indirect(reflect.ValueOf(obj)).Interface()
	return reflect.DeepEqual(indirect, zeroValue)
}
