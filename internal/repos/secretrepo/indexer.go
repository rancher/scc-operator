package secretrepo

import (
	"github.com/rancher-sandbox/scc-operator/internal/consts"
	corev1 "k8s.io/api/core/v1"
)

const (
	IndexSecretsByPath    = "scc.io/setting-by-namespace-and-name"
	IndexSecretsBySccHash = "scc.io/secret-refs-by-scc-hash"
)

func (r *SecretRepository) InitIndexers() {
	r.Cache.AddIndexer(
		IndexSecretsByPath,
		secretByPath,
	)

	r.Cache.AddIndexer(
		IndexSecretsBySccHash,
		secretToHash,
	)

}

func secretByPath(obj *corev1.Secret) ([]string, error) {
	// TODO: SCC namespace should be more configurable globally
	if obj.GetNamespace() != consts.DefaultSCCNamespace {
		return nil, nil
	}

	return []string{obj.GetNamespace() + "/" + obj.GetName()}, nil
}

func secretToHash(secret *corev1.Secret) ([]string, error) {
	if secret == nil {
		return nil, nil
	}

	hash, ok := secret.Labels[consts.LabelSccHash]
	if !ok {
		return []string{}, nil
	}
	return []string{hash}, nil
}

func (r *SecretRepository) GetBySccContentHash(contentHash string) ([]*corev1.Secret, error) {
	return r.Cache.GetByIndex(IndexSecretsBySccHash, contentHash)
}
