package controllers

import (
	"context"

	"github.com/rancher/wrangler/v3/pkg/generic"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

// ScopeFunc is a function that determines whether a scoped handler should trigger
type ScopeFunc func(key string, obj runtime.Object) (bool, error)

func inExpectedNamespace(obj runtime.Object, namespace string) bool {
	metadata, err := meta.Accessor(obj)
	if err != nil {
		return false
	}

	return metadata.GetNamespace() == namespace
}

func scopedOnChange[T generic.RuntimeMetaObject](ctx context.Context, name string, inScopeFunc ScopeFunc, c generic.ControllerMeta, sync generic.ObjectHandler[T]) {
	onChangeHandler := generic.FromObjectHandlerToHandler(sync)
	c.AddGenericHandler(ctx, name, func(key string, obj runtime.Object) (runtime.Object, error) {
		isInScope, err := inScopeFunc(key, obj)
		if err != nil || !isInScope {
			return obj, err
		}
		return onChangeHandler(key, obj)
	})
}

// TODO(wrangler/v4): revert to use OnRemove when it supports options (https://github.com/rancher/wrangler/pull/472).
func scopedOnRemove[T generic.RuntimeMetaObject](ctx context.Context, name string, inScopeFunc ScopeFunc, c generic.ControllerMeta, sync generic.ObjectHandler[T]) {
	onRemoveHandler := generic.NewRemoveHandler(name, c.Updater(), generic.FromObjectHandlerToHandler(sync))
	c.AddGenericHandler(ctx, name, func(key string, obj runtime.Object) (runtime.Object, error) {
		isInScope, err := inScopeFunc(key, obj)
		if err != nil || !isInScope {
			return obj, err
		}

		return onRemoveHandler(key, obj)
	})
}
