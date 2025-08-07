package v1

import (
	"github.com/rancher/wrangler/v3/pkg/genericcondition"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.secretType`
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type SecretRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretRequestSpec   `json:"spec,omitempty"`
	Status SecretRequestStatus `json:"status,omitempty"`
}

// SecretRequestSpec defines the secret type being requested, and the target where the secret will be created
type SecretRequestSpec struct {
	SecretType      string                  `json:"secretType"` // This is directly tied to instances of secrets that are registered.
	TargetSecretRef *corev1.SecretReference `json:"targetSecretRef"`
}

type SecretRequestStatus struct {
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []genericcondition.GenericCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// +optional
	LastSyncTS *metav1.Time `json:"lastSyncTS"`
}
