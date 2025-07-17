package controllers

import (
	"github.com/rancher-sandbox/scc-operator/internal/consts"
	v1 "github.com/rancher-sandbox/scc-operator/pkg/apis/scc.cattle.io/v1"
)

const (
	IndexRegistrationsBySccHash  = "scc.io/reg-refs-by-scc-hash"
	IndexRegistrationsByNameHash = "scc.io/reg-refs-by-name-hash"
)

func (h *handler) initIndexers() {
	h.registrationCache.AddIndexer(
		IndexRegistrationsBySccHash,
		h.registrationToHash,
	)
	h.registrationCache.AddIndexer(
		IndexRegistrationsByNameHash,
		h.registrationToNameHash,
	)
}

func (h *handler) registrationToHash(reg *v1.Registration) ([]string, error) {
	if reg == nil {
		return []string{}, nil
	}

	hash, ok := reg.Labels[consts.LabelSccHash]
	if !ok {
		return []string{}, nil
	}
	return []string{hash}, nil
}

func (h *handler) registrationToNameHash(reg *v1.Registration) ([]string, error) {
	if reg == nil {
		return []string{}, nil
	}

	hash, ok := reg.Labels[consts.LabelNameSuffix]
	if !ok {
		return []string{}, nil
	}
	return []string{hash}, nil
}
