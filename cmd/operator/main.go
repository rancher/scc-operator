package main

import (
	"context"
	"flag"
	"fmt"

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

func getOperatorMetadata() *types.OperatorMetadata {
	return &types.OperatorMetadata{
		Version:   version.Version,
		GitCommit: version.GitCommit,
		BuildDate: version.Date,
	}
}

func setupCli(ctx context.Context) *config.OperatorSettings {
	flag.StringVar(&config.LogFormat.FlagValue, "log-format", "", "Set the log format.")
	flag.StringVar(&config.LogLevel.FlagValue, "log-level", "", "Set the logging level.")
	flag.StringVar(&config.Kubeconfig.FlagValue, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&config.OperatorName.FlagValue, "operator-name", "", fmt.Sprintf("Name of the operator. Defaults to %s when unset.", consts.DefaultOperatorName))
	flag.StringVar(&config.OperatorNamespace.FlagValue, "operator-namespace", "", "The namespace where the operator is deployed.")
	flag.StringVar(&config.LeaseNamespace.FlagValue, "lease-namespace", "", "The namespace where the operator lease lives.")
	flag.BoolVar(&config.Debug.FlagValue, "debug", false, "Enable debug logging.")
	flag.BoolVar(&config.Trace.FlagValue, "trace", false, "Enable trace logging.")

	// SCC Product Config flags
	// TODO: These are temporary workaround for a better product universal mechanism
	// The future system must be one that allows one SCC operator to do many products via multiple registrations
	flag.StringVar(&config.Product.FlagValue, "product", "", "The product name that the operator is managing.")
	flag.StringVar(&config.ProductVersion.FlagValue, "product-version", "", "The version of the product to use.")

	flag.Parse()

	appConfig, err := config.LoadInitialConfig(ctx)
	if err != nil {
		logrus.Fatal(err)
	}

	rootLog.SetupLogging(appConfig.LogLevel, appConfig.LogFormat)
	log.AddDefaultOpts(
		rootLog.WithOperatorName(appConfig.OperatorName),
		rootLog.WithOperatorNamespace(appConfig.SystemNamespace),
	)
	logger = log.NewLog()

	return appConfig
}

func main() {
	ctx := signals.SetupSignalContext()
	operatorSettings := setupCli(ctx)

	logger.Infof("Starting %s version %s (%s) [built at %s]", consts.AppName, version.Version, version.GitCommit, version.Date)
	logger.Tracef("%+v", operatorSettings)
	restKubeConfig, err := kubeconfig.GetNonInteractiveClientConfig(operatorSettings.Kubeconfig).ClientConfig()
	if err != nil {
		if operatorSettings.Kubeconfig == "" {
			logger.Warn("If outside of cluster --kubeconfig is required")
		}
		logger.Fatalf("failed to find kubeconfig: %v", err)
	}

	// TODO: consider how SCC DevMode vs Rancher Dev mode should work with new system
	initializer.DevMode.Set(operatorSettings.CattleDevMode)
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
