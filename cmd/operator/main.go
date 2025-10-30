package main

import (
	"context"
	"fmt"

	"github.com/rancher/scc-operator/internal/config"
	"github.com/rancher/wrangler/v3/pkg/kubeconfig"
	"github.com/rancher/wrangler/v3/pkg/signals"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
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
	pflag.StringVar(&config.LogFormat.FlagValue, "log-format", "", "Set the log format.")
	pflag.StringVar(&config.LogLevel.FlagValue, "log-level", "", "Set the logging level.")
	pflag.StringVar(&config.Kubeconfig.FlagValue, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	pflag.StringVar(&config.OperatorName.FlagValue, "operator-name", "", fmt.Sprintf("Name of the operator. Defaults to %s when unset.", consts.DefaultOperatorName))
	pflag.StringVar(&config.OperatorNamespace.FlagValue, "operator-namespace", "", "The namespace where the operator is deployed.")
	pflag.StringVar(&config.LeaseNamespace.FlagValue, "lease-namespace", "", "The namespace where the operator lease lives.")
	pflag.BoolVar(&config.Debug.FlagValue, "debug", false, "Enable debug logging.")
	pflag.BoolVar(&config.Trace.FlagValue, "trace", false, "Enable trace logging.")
	pflag.Parse()

	flagSet := pflag.CommandLine
	appConfig, err := config.LoadInitialConfig(ctx, flagSet)
	if err != nil {
		logrus.Fatal(fmt.Errorf("error loading scc-operator config: %w", err))
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

	initializer.DevMode.Set(operatorSettings.DevMode)
	initializer.RancherDevMode.Set(operatorSettings.CattleDevMode)
	logger.Debugf(
		"Launching scc-operator; SCC Dev Mode: `%v`, Cattle Dev Mode: `%v`",
		initializer.DevMode.Get(),
		initializer.RancherDevMode.Get(),
	)

	if operatorSettings.DevMode {
		logger.Warn("with DevMode enabled log level will be forced to at least debug.")
		currentLevel := rootLog.GetLogLevel()
		if currentLevel < logrus.DebugLevel {
			rootLog.SetLogLevel(logrus.DebugLevel)
			logger = log.NewLog()
		}
	}

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

	// TODO(rancher-bias): this is rancher specific logic
	if metricErr := sccOperatorStarter.EnsureMetricsSecretRequest(ctx, runOptions.SystemNamespace()); metricErr != nil {
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
