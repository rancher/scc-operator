package controllers

import (
	"context"
	"github.com/rancher/wrangler/v3/pkg/generic"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

func inExpectedNamespace(obj runtime.Object, namespace string) bool {
	metadata, err := meta.Accessor(obj)
	if err != nil {
		return false
	}

	return metadata.GetNamespace() == namespace
}

func namespaceScopedCondition(namespace string) func(obj runtime.Object) bool {
	return func(obj runtime.Object) bool { return inExpectedNamespace(obj, namespace) }
}

func scopedOnChange[T generic.RuntimeMetaObject](ctx context.Context, name, namespace string, c generic.ControllerMeta, sync generic.ObjectHandler[T]) {
	condition := namespaceScopedCondition(namespace)
	onChangeHandler := generic.FromObjectHandlerToHandler(sync)
	c.AddGenericHandler(ctx, name, func(key string, obj runtime.Object) (runtime.Object, error) {
		if condition(obj) {
			return onChangeHandler(key, obj)
		}
		return obj, nil
	})
}

// TODO(wrangler/v4): revert to use OnRemove when it supports options (https://github.com/rancher/wrangler/pull/472).
func scopedOnRemove[T generic.RuntimeMetaObject](ctx context.Context, name, namespace string, c generic.ControllerMeta, sync generic.ObjectHandler[T]) {
	condition := namespaceScopedCondition(namespace)
	onRemoveHandler := generic.NewRemoveHandler(name, c.Updater(), generic.FromObjectHandlerToHandler(sync))
	c.AddGenericHandler(ctx, name, func(key string, obj runtime.Object) (runtime.Object, error) {
		if condition(obj) {
			return onRemoveHandler(key, obj)
		}
		return obj, nil
	})
}
