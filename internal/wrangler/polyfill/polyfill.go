package polyfill

import (
	"context"
	"strings"

	"github.com/rancher/wrangler/v3/pkg/generic"
	"k8s.io/apimachinery/pkg/runtime"
)

// ScopeFunc is a function that determines whether a scoped handler should trigger
type ScopeFunc func(key string, obj runtime.Object) (bool, error)

func InExpectedNamespace(nameIn string, _ runtime.Object, namespace string) bool {
	var namespaceIn string
	if strings.Contains(nameIn, "/") {
		parts := strings.Split(nameIn, "/")
		namespaceIn = parts[0]
	} else {
		namespaceIn = nameIn
	}

	return namespaceIn == namespace
}

func ScopedOnChange[T generic.RuntimeMetaObject](ctx context.Context, name string, inScopeFunc ScopeFunc, c generic.ControllerMeta, sync generic.ObjectHandler[T]) {
	onChangeHandler := generic.FromObjectHandlerToHandler(sync)
	c.AddGenericHandler(ctx, name, func(key string, obj runtime.Object) (runtime.Object, error) {
		isInScope, err := inScopeFunc(key, obj)
		if err != nil || !isInScope {
			return obj, err
		}
		return onChangeHandler(key, obj)
	})
}

// TODO(wrangler/v4): revert to use ScopedOnRemove when it supports options (https://github.com/rancher/wrangler/pull/472).
func ScopedOnRemove[T generic.RuntimeMetaObject](ctx context.Context, name string, inScopeFunc ScopeFunc, c generic.ControllerMeta, sync generic.ObjectHandler[T]) {
	onRemoveHandler := generic.NewRemoveHandler(name, c.Updater(), generic.FromObjectHandlerToHandler(sync))
	c.AddGenericHandler(ctx, name, func(key string, obj runtime.Object) (runtime.Object, error) {
		isInScope, err := inScopeFunc(key, obj)
		if err != nil || !isInScope {
			return obj, err
		}

		return onRemoveHandler(key, obj)
	})
}
