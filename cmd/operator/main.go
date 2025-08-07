package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/rancher/wrangler/v3/pkg/kubeconfig"
	"github.com/rancher/wrangler/v3/pkg/signals"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"

	"github.com/rancher/scc-operator/cmd/operator/version"
	"github.com/rancher/scc-operator/internal/consts"
	"github.com/rancher/scc-operator/internal/initializer"
	rootLog "github.com/rancher/scc-operator/internal/log"
	"github.com/rancher/scc-operator/internal/types"
	"github.com/rancher/scc-operator/pkg/operator"
	"github.com/rancher/scc-operator/pkg/util/log"
)

var (
	KubeConfig     string
	LogFormat      string
	Debug          bool
	Trace          bool
	SCCNamespace   string
	LeaseNamespace string
	OperatorName   string
	logger         rootLog.StructuredLogger
)

func init() {
	flag.StringVar(&LogFormat, "log-format", string(rootLog.DefaultFormat), "Set the log format")

	kubeConfigEnv := os.Getenv("KUBECONFIG")
	flag.StringVar(&KubeConfig, "kubeconfig", kubeConfigEnv, "Path to a kubeconfig. Only required if out-of-cluster.")

	operatorName := os.Getenv("SCC_OPERATOR_NAME")
	if operatorName == "" {
		operatorName = consts.DefaultOperatorName
	}
	operatorNameUsage := fmt.Sprintf("Name of the operator. Defaults to %s", consts.DefaultOperatorName)
	flag.StringVar(&OperatorName, "operator-name", operatorName, operatorNameUsage)

	flag.BoolVar(&Debug, "debug", false, "Enable debug logging.")
	flag.BoolVar(&Trace, "trace", false, "Enable trace logging.")

	rootLog.ParseAndSetLogFormatFromString(LogFormat)
	if Debug {
		rootLog.SetLogLevel(logrus.DebugLevel)
		logrus.Debugf("Loglevel set to [%v]", logrus.DebugLevel)
	}
	if Trace {
		rootLog.SetLogLevel(logrus.TraceLevel)
		logrus.Tracef("Loglevel set to [%v]", logrus.TraceLevel)
	}

	flag.Parse()
	SCCNamespace = os.Getenv("SCC_SYSTEM_NAMESPACE")
	if SCCNamespace == "" {
		SCCNamespace = consts.DefaultSCCNamespace
	}

	LeaseNamespace = os.Getenv("SCC_LEASE_NAMESPACE")

	log.AddDefaultOpts(rootLog.WithOperatorName(OperatorName))
	logger = log.NewLog()
}

func main() {
	logger.Infof("Starting %s version %s (%s) [built at %s]", consts.AppName, version.Version, version.GitCommit, version.Date)
	ctx := signals.SetupSignalContext()
	restKubeConfig, err := kubeconfig.GetNonInteractiveClientConfig(KubeConfig).ClientConfig()
	if err != nil {
		if KubeConfig == "" {
			logger.Warn("If outside of cluster --kubeconfig is required")
		}
		logger.Fatalf("failed to find kubeconfig: %v", err)
	}

	dm := os.Getenv("CATTLE_DEV_MODE")
	initializer.DevMode.Set(dm != "")
	runOptions := types.RunOptions{
		Logger:         logger,
		OperatorName:   OperatorName,
		SccNamespace:   SCCNamespace,
		LeaseNamespace: LeaseNamespace,
	}

	if err := run(ctx, restKubeConfig, runOptions); err != nil {
		logger.Fatal(err)
	}
}

func run(ctx context.Context, restKubeConfig *rest.Config, runOptions types.RunOptions) error {
	logger.Debugf("Setting up client for %s...", SCCNamespace)
	logger.Debugf("Run options: %v", runOptions)

	sccOperatorStarter, err := operator.New(ctx, restKubeConfig, runOptions)
	if err != nil {
		logger.Errorf("Error creating operator: %v", err)
		return err
	}

	if metricErr := sccOperatorStarter.EnsureMetricsSecretRequest(ctx); metricErr != nil {
		logger.Errorf("Error ensuring metrics secret request: %v", metricErr)
		return metricErr
	}

	if runErr := sccOperatorStarter.Run(); runErr != nil {
		logger.Errorf("Error running operator: %v", runErr)
		return runErr
	}
	<-ctx.Done()
	return nil
}
