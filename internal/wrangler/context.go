package wrangler

import (
	"context"
	"fmt"
	"sync"

	lasso "github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/lasso/pkg/mapper"
	"github.com/rancher/scc-operator/internal/consts"
	"github.com/rancher/scc-operator/internal/logging"
	"github.com/rancher/scc-operator/internal/rancher/settings"
	v1 "github.com/rancher/scc-operator/pkg/apis/scc.cattle.io/v1"
	v1core "github.com/rancher/wrangler/v3/pkg/generated/controllers/core"
	corev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/generic"
	"github.com/rancher/wrangler/v3/pkg/leader"
	"github.com/rancher/wrangler/v3/pkg/schemes"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/rancher/scc-operator/internal/repos/secretrepo"
	sccControllers "github.com/rancher/scc-operator/pkg/generated/controllers/scc.cattle.io"
	sccv1 "github.com/rancher/scc-operator/pkg/generated/controllers/scc.cattle.io/v1"
)

var (
	localSchemeBuilder = runtime.SchemeBuilder{
		v1.AddToScheme,
		scheme.AddToScheme,
	}
	AddToScheme = localSchemeBuilder.AddToScheme
	Scheme      = runtime.NewScheme()
)

func init() {
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
	utilruntime.Must(AddToScheme(Scheme))
	utilruntime.Must(schemes.AddToScheme(Scheme))
}

type MiniContext struct {
	RESTConfig *rest.Config

	Dynamic           *dynamic.DynamicClient
	ControllerFactory controller.SharedControllerFactory
	SharedFactory     lasso.SharedClientFactory
	K8sClient         *kubernetes.Clientset
	Mapper            meta.RESTMapper
	ClientSet         *clientset.Clientset

	Core    corev1.Interface
	SCC     sccv1.Interface
	Secrets *secretrepo.SecretRepository

	Settings *settings.SettingReader

	leadership     *leader.Manager
	controllerLock *sync.Mutex
}

func enableProtobuf(cfg *rest.Config) *rest.Config {
	cpy := rest.CopyConfig(cfg)
	cpy.AcceptContentTypes = "application/vnd.kubernetes.protobuf, application/json"
	cpy.ContentType = "application/json"
	return cpy
}

func NewWranglerMiniContext(_ context.Context, restConfig *rest.Config, systemNamespace, leaseNamespace string) (MiniContext, error) {
	logging.Logger.Debugf("Lease namespace is %s", leaseNamespace)
	controllerFactory, err := controller.NewSharedControllerFactoryFromConfig(enableProtobuf(restConfig), Scheme)
	if err != nil {
		return MiniContext{}, err
	}

	clientSet, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return MiniContext{}, fmt.Errorf("error getting clientSet: %s", err.Error())
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return MiniContext{}, fmt.Errorf("error generating dynamic client: %s", err.Error())
	}

	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return MiniContext{}, fmt.Errorf("error getting kubernetes client: %s", err.Error())
	}

	restMapper, err := mapper.New(restConfig)
	if err != nil {
		return MiniContext{}, fmt.Errorf("error building rest mapper: %s", err.Error())
	}

	sharedClientFactory, err := lasso.NewSharedClientFactoryForConfig(restConfig)
	if err != nil {
		return MiniContext{}, fmt.Errorf("error generating shared client factory: %s", err.Error())
	}

	opts := &generic.FactoryOptions{
		SharedControllerFactory: controllerFactory,
	}

	coreF, err := v1core.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return MiniContext{}, fmt.Errorf("error building core sample controllers: %s", err.Error())
	}

	sccFactory, err := sccControllers.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return MiniContext{}, err
	}

	coreInterface := coreF.Core().V1()
	secretRepo := secretrepo.NewSecretRepository(systemNamespace, coreInterface.Secret(), coreInterface.Secret().Cache())

	k8s, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return MiniContext{}, err
	}

	// By default, the `leaseNamespace` will be empty which defaults to `kube-system`.
	// If there are multiple SCC operator instances, only one will have controller leases at a time.
	logging.Logger.Debugf("Creating new leader manager in namespace %s", leaseNamespace)
	leadership := leader.NewManager(leaseNamespace, "scc-controllers", k8s)

	return MiniContext{
		RESTConfig: restConfig,

		ClientSet:         clientSet,
		ControllerFactory: controllerFactory,
		Dynamic:           dynamicClient,
		K8sClient:         k8sClient,
		Mapper:            restMapper,
		SharedFactory:     sharedClientFactory,

		Core:    coreInterface,
		SCC:     sccFactory.Scc().V1(),
		Secrets: secretRepo,

		Settings: settings.NewSettingReader(dynamicClient),

		leadership:     leadership,
		controllerLock: &sync.Mutex{},
	}, nil
}

func (c *MiniContext) Start(ctx context.Context) error {
	c.controllerLock.Lock()
	defer c.controllerLock.Unlock()

	logging.Logger.Debug("Starting controller factory")
	if err := c.ControllerFactory.Start(ctx, consts.OperatorWorkerThreads); err != nil {
		return err
	}
	logging.Logger.Debug("Starting leadership manager")
	c.leadership.Start(ctx)
	return nil
}

func (c *MiniContext) OnLeader(f func(ctx context.Context) error) {
	c.leadership.OnLeader(func(ctx context.Context) error {
		logging.Logger.Debug("Acquired leadership, running controller")
		return f(ctx)
	})
}
