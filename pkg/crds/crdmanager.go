package crds

import (
	"context"
	"embed"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"time"

	"github.com/rancher/wrangler/v3/pkg/crd"
	"github.com/rancher/wrangler/v3/pkg/yaml"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	clientv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/rancher/scc-operator/internal/consts"
	"github.com/rancher/scc-operator/internal/types"
)

const (
	// readyDuration time to wait for CRDs to be ready.
	readyDuration = time.Minute * 1

	crdKind = "CustomResourceDefinition"
)

var (
	//go:embed yaml
	crdFS embed.FS

	errDuplicate = fmt.Errorf("duplicate CRD")
)

type CRDTracker interface {
	List() []*apiextv1.CustomResourceDefinition
	EnsureRequired(ctx context.Context) error
	Ensure(ctx context.Context, crdNames []string) error
}

type CrdManager struct {
	crdClient      *clientv1.CustomResourceDefinitionInterface
	managedByValue string
	allCRDs        map[string]*apiextv1.CustomResourceDefinition
}

func (c *CrdManager) List() []*apiextv1.CustomResourceDefinition {
	return slices.Collect(maps.Values(c.allCRDs))
}

func (c *CrdManager) GetItems(names []string) []*apiextv1.CustomResourceDefinition {
	filteredCRDs := make(map[string]*apiextv1.CustomResourceDefinition)
	for key, CRD := range c.allCRDs {
		if slices.Contains(names, key) {
			if CRD.Labels == nil {
				CRD.Labels = map[string]string{}
			}
			CRD.Labels[consts.LabelK8sManagedBy] = c.managedByValue

			filteredCRDs[key] = CRD
		}
	}

	return slices.Collect(maps.Values(filteredCRDs))
}

func (c *CrdManager) EnsureRequired(ctx context.Context) error {
	return c.Ensure(ctx, RequiredCRDs())
}

func (c *CrdManager) Ensure(ctx context.Context, crdNames []string) error {
	allCRDs := c.GetItems(crdNames)

	// Create scc-operator owner label selector (specific to name in use at runtime)
	ownedByOperator, err := labels.NewRequirement(consts.LabelK8sManagedBy, selection.Equals, []string{c.managedByValue})
	if err != nil {
		return fmt.Errorf("failed to create crd label selector: %w", err)
	}
	selector := labels.NewSelector().Add(*ownedByOperator)

	err = crd.BatchCreateCRDs(ctx, *c.crdClient, selector, readyDuration, allCRDs)
	if err != nil {
		return fmt.Errorf("failed to create CRDs: %w", err)
	}

	return nil
}

var _ CRDTracker = &CrdManager{}

func NewCRDManager(options types.RunOptions, crdClient clientv1.CustomResourceDefinitionInterface) CRDTracker {
	allCRDs, err := crdsFromDir("yaml")
	if err != nil {
		return nil
	}

	return &CrdManager{
		crdClient:      &crdClient,
		managedByValue: options.OperatorName,
		allCRDs:        allCRDs,
	}
}

// crdsFromDir recursively traverses the embedded yaml directory and find all CRD yamls.
// cribbed from https://github.com/rancher/rancher/blob/main/pkg/crds/crds.go
func crdsFromDir(dirName string) (map[string]*apiextv1.CustomResourceDefinition, error) {
	// read all entries in the embedded directory
	crdFiles, err := crdFS.ReadDir(dirName)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded dir '%s': %w", dirName, err)
	}

	allCRDs := map[string]*apiextv1.CustomResourceDefinition{}
	for _, dirEntry := range crdFiles {
		fullPath := filepath.Join(dirName, dirEntry.Name())
		if dirEntry.IsDir() {
			// if the entry is the dir recurse into that folder to get all crds
			subCRDs, err := crdsFromDir(fullPath)
			if err != nil {
				return nil, err
			}
			for k, v := range subCRDs {
				if _, ok := allCRDs[k]; ok {
					return nil, fmt.Errorf("%w for '%s", errDuplicate, k)
				}
				allCRDs[k] = v
			}
			continue
		}

		// read the file and convert it to a crd object
		file, err := crdFS.Open(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open embedded file '%s': %w", fullPath, err)
		}
		crdObjs, err := yaml.UnmarshalWithJSONDecoder[*apiextv1.CustomResourceDefinition](file)
		if err != nil {
			return nil, fmt.Errorf("failed to convert embedded file '%s' to yaml: %w", fullPath, err)
		}
		for _, crdObj := range crdObjs {
			if crdObj.Kind != crdKind {
				// if the yaml is not a CRD return an error
				return nil, fmt.Errorf("decoded object is not '%s' instead found Kind='%s'", crdKind, crdObj.Kind)
			}
			if _, ok := allCRDs[crdObj.Name]; ok {
				return nil, fmt.Errorf("%w for '%s", errDuplicate, crdObj.Name)
			}
			allCRDs[crdObj.Name] = crdObj
		}
	}
	return allCRDs, nil
}
