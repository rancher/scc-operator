package settings

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestNewSettingReader_FunctionalWithDynamicFake(t *testing.T) {
	asserts := assert.New(t)

	scheme := runtime.NewScheme()
	gv := schema.GroupVersion{Group: RancherMgmtGroup, Version: RancherMgmtVersion}
	scheme.AddKnownTypeWithName(gv.WithKind("Setting"), &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(gv.WithKind("SettingList"), &unstructured.UnstructuredList{})

	// Seed a few Setting objects
	name1 := "my-setting"
	obj1 := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": gv.String(),
		"kind":       "Setting",
		"metadata": map[string]interface{}{
			"name": name1,
		},
		"default": "A",
		"value":   "B",
	}}
	name2 := "unchanged-setting"
	obj2 := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": gv.String(),
		"kind":       "Setting",
		"metadata": map[string]interface{}{
			"name": name2,
		},
		"default": "still-the-default",
	}}

	fake := dynamicfake.NewSimpleDynamicClient(scheme, obj1, obj2)
	reader := NewSettingReader(fake)

	ctx := context.Background()
	asserts.True(reader.Has(ctx, name1))
	asserts.False(reader.Has(ctx, "does-not-exist"))

	ps, err := reader.Get(ctx, name1)
	asserts.Nil(err)
	asserts.NotNil(ps)
	asserts.Equal("A", ps.Default)
	asserts.Equal("B", ps.Value)
	asserts.Equal("B", ps.Get())

	ps, err = reader.Get(ctx, name2)
	asserts.Nil(err)
	asserts.NotNil(ps)
	asserts.Equal("still-the-default", ps.Default)
	asserts.Empty(ps.Value)
	asserts.Equal("still-the-default", ps.Get())
}

func TestSettingReader_Get_EdgeCases(t *testing.T) {
	asserts := assert.New(t)

	scheme := runtime.NewScheme()
	gv := schema.GroupVersion{Group: RancherMgmtGroup, Version: RancherMgmtVersion}
	scheme.AddKnownTypeWithName(gv.WithKind("Setting"), &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(gv.WithKind("SettingList"), &unstructured.UnstructuredList{})

	// onlyDefault: missing `value` should make Get() return `default`
	onlyDefault := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": gv.String(),
		"kind":       "Setting",
		"metadata": map[string]interface{}{
			"name": "only-default",
		},
		"default": "fallback",
	}}

	// badType: `default` has wrong type, should trigger conversion error in Get
	badType := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": gv.String(),
		"kind":       "Setting",
		"metadata": map[string]interface{}{
			"name": "bad-type",
		},
		"default": 123.0, // JSON number (float64) -> conversion to string should fail
	}}

	fake := dynamicfake.NewSimpleDynamicClient(scheme, onlyDefault, badType)
	reader := NewSettingReader(fake)
	ctx := context.Background()

	// Non-existing -> error
	ps, err := reader.Get(ctx, "does-not-exist")
	asserts.Nil(ps)
	asserts.NotNil(err)

	// Missing value -> default used
	ps, err = reader.Get(ctx, "only-default")
	asserts.Nil(err)
	asserts.NotNil(ps)
	asserts.Empty(ps.Value)
	asserts.Equal("fallback", ps.Get())

	// Bad type -> conversion error
	ps, err = reader.Get(ctx, "bad-type")
	asserts.Nil(ps)
	asserts.NotNil(err)
}
