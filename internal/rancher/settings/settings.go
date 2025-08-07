package settings

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type SettingReader struct {
	scopedDynamicClient dynamic.NamespaceableResourceInterface
}

func NewSettingReader(dynamicClient dynamic.Interface) *SettingReader {
	return &SettingReader{
		scopedDynamicClient: dynamicClient.Resource(rancherSettingGVR()),
	}
}

const (
	RancherMgmtGroup           = "management.cattle.io"
	RancherMgmtVersion         = "v3"
	RancherMgmtSettingResource = "settings"
)

func rancherSettingGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    RancherMgmtGroup,
		Version:  RancherMgmtVersion,
		Resource: RancherMgmtSettingResource,
	}
}

func (s *SettingReader) Get(ctx context.Context, name string) (*ProtoSetting, error) {
	var protoSetting ProtoSetting
	item, err := s.scopedDynamicClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if convertErr := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &protoSetting); convertErr != nil {
		return nil, fmt.Errorf("failed to convert unstructured item to proto setting: %v", convertErr)
	}

	return &protoSetting, nil
}

func (s *SettingReader) Has(ctx context.Context, name string) bool {
	_, err := s.scopedDynamicClient.Get(ctx, name, metav1.GetOptions{})
	return err == nil
}

type ProtoSetting struct {
	Name    string `json:"metadata.name"`
	Default string `json:"default"`
	Value   string `json:"value"`
}

func (ps *ProtoSetting) Get() string {
	if ps.Value == "" {
		return ps.Default
	}

	return ps.Value
}
