package controllers

import (
	"context"

	"github.com/rancher/wrangler/v3/pkg/relatedresource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/rancher/scc-operator/internal/consts"
)

// initResolvers creates a secret watcher to check for SCC entrypoint secrets
func (h *handler) initResolvers(ctx context.Context) {
	relatedresource.Watch(
		ctx,
		"watch-scc-secret-entrypoint",
		h.resolveEntrypointSecret,
		h.secretRepo.Controller,
	)
}

func (h *handler) resolveEntrypointSecret(namespace, name string, obj runtime.Object) ([]relatedresource.Key, error) {
	var relatedKeys []relatedresource.Key
	if namespace != h.options.SystemNamespace() {
		return relatedKeys, nil
	}
	if name != consts.ResourceSCCEntrypointSecretName {
		return relatedKeys, nil
	}

	// Only handle secrets - objects of other types ignored by this watcher.
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return relatedKeys, nil
	}

	curHash, ok := secret.GetLabels()[consts.LabelSccHash]
	if !ok {
		// TODO: is this a chance to better handle new/modified registrations?
		h.log.Warnf("failed to find hash for secret %s/%s", namespace, name)
		return relatedKeys, nil
	}
	// TODO: rework indexers / resolvers and potentially remove that pattern
	defer func() {
		if r := recover(); r != nil {
			h.log.Errorf("recovered from panic in secret %s/%s with hash %s: %v", namespace, name, curHash, r)
		}
	}()
	regs, err := h.registrationCache.GetByIndex(IndexRegistrationsBySccHash, curHash)
	if err != nil {
		return relatedKeys, err
	}

	h.log.Infof("resolved entrypoint secret to : %d registrations", len(regs))
	for _, reg := range regs {
		if reg == nil {
			continue
		}
		relatedKeys = append(relatedKeys, relatedresource.Key{
			Name: reg.GetName(),
		})
	}
	return relatedKeys, nil
}
