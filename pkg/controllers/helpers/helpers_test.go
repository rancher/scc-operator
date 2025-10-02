package helpers

import (
	"testing"

	"github.com/rancher/scc-operator/internal/consts"
	v1 "github.com/rancher/scc-operator/pkg/apis/scc.cattle.io/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_isEmptyObject(t *testing.T) {
	cases := map[string]struct {
		obj     interface{}
		isEmpty bool
	}{
		"nil object": {
			obj:     nil,
			isEmpty: true,
		},
		"empty secret": {
			obj:     &corev1.Secret{},
			isEmpty: true,
		},
		"empty object": {
			obj:     &v1.Registration{},
			isEmpty: true,
		},
		"non-empty object": {
			obj: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						consts.LabelK8sManagedBy: "test-manager",
					},
				},
			},
			isEmpty: false,
		},
		"non-empty object 2": {
			obj: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-manager",
				},
			},
			isEmpty: false,
		},
		"empty reg object with explicit nils": {
			obj: &v1.Registration{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      nil,
					Annotations: nil,
				},
			},
			isEmpty: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			isEmpty := isEmptyObject(tc.obj)
			assert.Equal(t, tc.isEmpty, isEmpty)
		})
	}
}
