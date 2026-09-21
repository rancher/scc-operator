package controllers

import (
	"testing"

	"github.com/rancher/scc-operator/internal/initializer"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/scc-operator/internal/consts"
	v1 "github.com/rancher/scc-operator/pkg/apis/scc.cattle.io/v1"
)

func TestRegistrationFromSecret(t *testing.T) {
	initializer.DevMode.Set(true)

	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{},
		Data: map[string][]byte{
			consts.SecretKeyRegistrationCode: []byte("hello"),
			dataKeyRegistrationType:          []byte(v1.RegistrationModeOnline),
		},
	}

	params, err := extractRegistrationParamsFromSecret(sec, "testing")
	assert.NoError(t, err)

	assert.NotNil(t, params)
	assert.NotNil(t, params.contentHash)
	// valid hash

	seenHash := params.contentHash
	assert.Equal(t, len(seenHash), 32)

	sec.Data[consts.SecretKeyRegistrationCode] = []byte("world")
	params2, err := extractRegistrationParamsFromSecret(sec, "testing")
	assert.NoError(t, err)
	assert.NotNil(t, params2)
	assert.NotNil(t, params2.contentHash)
	assert.NotEqual(t, seenHash, params2.contentHash)

	seenHash = params2.contentHash
	sec.Data[dataKeyRegistrationType] = []byte(v1.RegistrationModeOffline)
	params3, err := extractRegistrationParamsFromSecret(sec, "testing")
	assert.NoError(t, err)
	assert.NotNil(t, params3)
	assert.NotNil(t, params3.contentHash)
	assert.NotEqual(t, seenHash, params3.contentHash)

	for _, label := range params.Labels() {
		assert.Less(t, len(label), 63)
	}

	for _, label := range params2.Labels() {
		assert.Less(t, len(label), 63)
	}

	for _, label := range params3.Labels() {
		assert.Less(t, len(label), 63)
	}
}

func TestParamsToRegSpecWithInstanceData(t *testing.T) {
	tests := []struct {
		name     string
		params   RegistrationParams
		validate func(t *testing.T, regSpec v1.RegistrationSpec)
	}{
		{
			name: "sets instance data secret ref when hasInstanceData is true",
			params: RegistrationParams{
				regType:     v1.RegistrationModeOnline,
				regURL:      "https://rmt.example.com",
				regCodeSecretRef: &corev1.SecretReference{
					Name:      "regcode-secret",
					Namespace: "test-namespace",
				},
				hasInstanceData: true,
				rmtInstanceDataSecretRef: &corev1.SecretReference{
					Name:      "instance-data-secret",
					Namespace: "test-namespace",
				},
			},
			validate: func(t *testing.T, regSpec v1.RegistrationSpec) {
				assert.NotNil(t, regSpec.RegistrationRequest.RegistrationInstanceDataSecretRef)
				assert.Equal(t, "instance-data-secret", regSpec.RegistrationRequest.RegistrationInstanceDataSecretRef.Name)
				assert.Equal(t, "test-namespace", regSpec.RegistrationRequest.RegistrationInstanceDataSecretRef.Namespace)
			},
		},
		{
			name: "does not set instance data secret ref when hasInstanceData is false",
			params: RegistrationParams{
				regType:     v1.RegistrationModeOnline,
				regURL:      "https://rmt.example.com",
				regCodeSecretRef: &corev1.SecretReference{
					Name:      "regcode-secret",
					Namespace: "test-namespace",
				},
				hasInstanceData: false,
			},
			validate: func(t *testing.T, regSpec v1.RegistrationSpec) {
				assert.Nil(t, regSpec.RegistrationRequest.RegistrationInstanceDataSecretRef)
			},
		},
		{
			name: "does not overwrite cert ref when setting instance data ref",
			params: RegistrationParams{
				regType:     v1.RegistrationModeOnline,
				regURL:      "https://rmt.example.com",
				regCodeSecretRef: &corev1.SecretReference{
					Name:      "regcode-secret",
					Namespace: "test-namespace",
				},
				hasRegURLCertData: true,
				regURLCertSecretRef: &corev1.SecretReference{
					Name:      "cert-secret",
					Namespace: "test-namespace",
				},
				hasInstanceData: true,
				rmtInstanceDataSecretRef: &corev1.SecretReference{
					Name:      "instance-data-secret",
					Namespace: "test-namespace",
				},
			},
			validate: func(t *testing.T, regSpec v1.RegistrationSpec) {
				assert.NotNil(t, regSpec.RegistrationRequest.RegistrationAPICertificateSecretRef)
				assert.Equal(t, "cert-secret", regSpec.RegistrationRequest.RegistrationAPICertificateSecretRef.Name)
				assert.NotNil(t, regSpec.RegistrationRequest.RegistrationInstanceDataSecretRef)
				assert.Equal(t, "instance-data-secret", regSpec.RegistrationRequest.RegistrationInstanceDataSecretRef.Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regSpec := paramsToRegSpec(tt.params)
			tt.validate(t, regSpec)
		})
	}
}

