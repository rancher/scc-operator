package generic

import (
	"github.com/rancher/wrangler/v3/pkg/generic"
	"k8s.io/apimachinery/pkg/runtime"
)

type RuntimeObjectRepo[T generic.RuntimeMetaObject, TList runtime.Object] struct {
	Controller generic.ControllerInterface[T, TList]
	Cache      generic.CacheInterface[T]
}

type NonNamespacedRuntimeObjectRepo[T generic.RuntimeMetaObject, TList runtime.Object] struct {
	Controller generic.NonNamespacedControllerInterface[T, TList]
	Cache      generic.NonNamespacedCacheInterface[T]
}

type RuntimeObjectRepository interface {
	InitIndexers()
}
