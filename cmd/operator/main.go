package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/rancher/scc-operator/internal/config"
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
	logger rootLog.StructuredLogger
)

func initialEnvValues() config.EnvVarsMap {
	return config.EnvVarsMap{
		config.KubeconfigEnv:         config.KubeconfigEnv.Get(),
		config.LogLevelEnv:           config.LogLevelEnv.Get(),
		config.LogFormatEnv:          config.LogFormatEnv.Get(),
		config.SCCOperatorNameEnv:    config.SCCOperatorNameEnv.Get(),
		config.SCCSystemNamespaceEnv: config.SCCSystemNamespaceEnv.Get(),
		config.SCCLeaseNamespaceEnv:  config.SCCLeaseNamespaceEnv.Get(),
		config.DebugEnv:              config.GetDebugEnv(),
		config.TraceEnv:              config.GetTraceEnv(),
	}
}

func getOperatorMetadata() *types.OperatorMetadata {
	return &types.OperatorMetadata{
		Version:   version.Version,
		GitCommit: version.GitCommit,
		BuildDate: version.Date,
	}
}

func setupCli(ctx context.Context) *config.OperatorSettings {
	var flags config.FlagValues
	flag.StringVar(&flags.LogFormat, "log-format", "", "Set the log format.")
	flag.StringVar(&flags.LogLevel, "log-level", "", "Set the logging level.")
	flag.StringVar(&flags.KubeconfigPath, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	operatorNameUsage := fmt.Sprintf("Name of the operator. Defaults to %s when unset.", consts.DefaultOperatorName)
	flag.StringVar(&flags.OperatorName, "operator-name", "", operatorNameUsage)
	flag.StringVar(&flags.OperatorNamespace, "operator-namespace", "", "The namespace where the operator is deployed.")
	flag.StringVar(&flags.LeaseNamespace, "lease-namespace", "", "The namespace where the operator lease lives.")
	flag.BoolVar(&flags.Debug, "debug", false, "Enable debug logging.")
	flag.BoolVar(&flags.Trace, "trace", false, "Enable trace logging.")
	flag.Parse()

	envValues := initialEnvValues()

	appConfig, err := config.LoadInitialConfig(ctx, &flags, envValues)
	if err != nil {
		logrus.Fatal(err)
	}

	rootLog.SetupLogging(appConfig.LogLevel, appConfig.LogFormat)
	log.AddDefaultOpts(
		rootLog.WithOperatorName(flags.OperatorName),
		rootLog.WithOperatorNamespace(appConfig.SystemNamespace),
	)
	logger = log.NewLog()

	return appConfig
}

func main() {
	ctx := signals.SetupSignalContext()
	operatorSettings := setupCli(ctx)

	logger.Infof("Starting %s version %s (%s) [built at %s]", consts.AppName, version.Version, version.GitCommit, version.Date)
	restKubeConfig, err := kubeconfig.GetNonInteractiveClientConfig(operatorSettings.Kubeconfig).ClientConfig()
	if err != nil {
		if operatorSettings.Kubeconfig == "" {
			logger.Warn("If outside of cluster --kubeconfig is required")
		}
		logger.Fatalf("failed to find kubeconfig: %v", err)
	}

	dm := os.Getenv("CATTLE_DEV_MODE")
	initializer.DevMode.Set(dm != "")
	logger.Debugf("Launching scc-operator with DevMode set to `%v`", initializer.DevMode.Get())

	runOptions := types.RunOptions{
		Logger:           logger,
		OperatorSettings: config.GetCurrentConfig(),
		OperatorName:     operatorSettings.OperatorName,
		DevMode:          initializer.DevMode.Get(),
		OperatorMetadata: *getOperatorMetadata(),
	}

	if err := run(ctx, restKubeConfig, runOptions); err != nil {
		logger.Fatal(err)
	}
}

func run(ctx context.Context, restKubeConfig *rest.Config, runOptions types.RunOptions) error {
	logger.Debugf("Setting up `%s` client for '%s' namespace", runOptions.OperatorSettings.OperatorName, runOptions.OperatorSettings.SystemNamespace)
	logger.Debugf("Run options: %v", runOptions)

	sccOperatorStarter, err := operator.New(ctx, restKubeConfig, runOptions)
	if err != nil {
		logger.Errorf("Error creating operator: %v", err)
		return err
	}

	if metricErr := sccOperatorStarter.EnsureMetricsSecretRequest(ctx, runOptions.OperatorSettings.SystemNamespace); metricErr != nil {
		logger.Errorf("Error ensuring metrics secret request: %v", metricErr)
		return metricErr
	}

	go sccOperatorStarter.StartMetricsAndHealthEndpoint()
	if runErr := sccOperatorStarter.Run(); runErr != nil {
		logger.Errorf("Error running operator: %v", runErr)
		return runErr
	}
	<-ctx.Done()
	return nil
}
