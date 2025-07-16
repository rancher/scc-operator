package wrangler

import (
	"fmt"
	lasso "github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/lasso/pkg/mapper"
	v1core "github.com/rancher/wrangler/v3/pkg/generated/controllers/core"
	corev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/generic"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/rancher-sandbox/scc-operator/pkg/generated/controllers/management.cattle.io"
	mgmtv3 "github.com/rancher-sandbox/scc-operator/pkg/generated/controllers/management.cattle.io/v3"
	"github.com/rancher-sandbox/scc-operator/pkg/generated/controllers/scc.cattle.io"
	sccv1 "github.com/rancher-sandbox/scc-operator/pkg/generated/controllers/scc.cattle.io/v1"
)

var (
	Scheme = runtime.NewScheme()
)

type MiniContext struct {
	RESTConfig    *rest.Config
	Mapper        meta.RESTMapper
	ClientSet     *clientset.Clientset
	K8sClient     *kubernetes.Clientset
	Dynamic       *dynamic.DynamicClient
	SharedFactory lasso.SharedClientFactory
	Core          corev1.Interface
	SCC           sccv1.Interface
	Mgmt          mgmtv3.Interface
}

func enableProtobuf(cfg *rest.Config) *rest.Config {
	cpy := rest.CopyConfig(cfg)
	cpy.AcceptContentTypes = "application/vnd.kubernetes.protobuf, application/json"
	cpy.ContentType = "application/json"
	return cpy
}

func NewWranglerMiniContext(kubeConfig *rest.Config) (MiniContext, error) {
	controllerFactory, err := controller.NewSharedControllerFactoryFromConfig(enableProtobuf(kubeConfig), Scheme)
	if err != nil {
		return MiniContext{}, err
	}

	opts := &generic.FactoryOptions{
		SharedControllerFactory: controllerFactory,
	}

	restmapper, err := mapper.New(kubeConfig)
	if err != nil {
		return MiniContext{}, fmt.Errorf("error building rest mapper: %s", err.Error())
	}

	clientSet, err := clientset.NewForConfig(kubeConfig)
	if err != nil {
		return MiniContext{}, fmt.Errorf("error getting clientSet: %s", err.Error())
	}

	k8sclient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return MiniContext{}, fmt.Errorf("error getting kubernetes client: %s", err.Error())
	}

	dynamicInterface, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		return MiniContext{}, fmt.Errorf("error generating dynamic client: %s", err.Error())
	}
	sharedClientFactory, err := lasso.NewSharedClientFactoryForConfig(kubeConfig)
	if err != nil {
		return MiniContext{}, fmt.Errorf("error generating shared client factory: %s", err.Error())
	}

	coreF, err := v1core.NewFactoryFromConfigWithOptions(kubeConfig, opts)
	if err != nil {
		return MiniContext{}, fmt.Errorf("error building core sample controllers: %s", err.Error())
	}

	sccFactory, err := scc.NewFactoryFromConfigWithOptions(kubeConfig, opts)
	if err != nil {
		return MiniContext{}, err
	}

	mgmtFactory, err := management.NewFactoryFromConfigWithOptions(kubeConfig, opts)
	if err != nil {
		return MiniContext{}, err
	}

	return MiniContext{
		RESTConfig:    kubeConfig,
		Mapper:        restmapper,
		ClientSet:     clientSet,
		K8sClient:     k8sclient,
		Dynamic:       dynamicInterface,
		SharedFactory: sharedClientFactory,
		Core:          coreF.Core().V1(),
		SCC:           sccFactory.Scc().V1(),
		Mgmt:          mgmtFactory.Management().V3(),
	}, nil
}
