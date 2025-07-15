package main

import (
	"flag"
	"github.com/rancher-sandbox/scc-operator/pkg/log"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/rancher/wrangler/v3/pkg/kubeconfig"
	"github.com/rancher/wrangler/v3/pkg/signals"

	"github.com/rancher-sandbox/scc-operator/cmd/operator/version"
	rootLog "github.com/rancher-sandbox/scc-operator/internal/log"
	"github.com/rancher-sandbox/scc-operator/internal/types"
	"github.com/rancher-sandbox/scc-operator/internal/util"
	"github.com/rancher-sandbox/scc-operator/pkg/operator"
)

var (
	KubeConfig   string
	LogFormat    string
	Debug        bool
	Trace        bool
	SCCNamespace string
	logger       rootLog.StructuredLogger
)

func init() {
	flag.StringVar(&LogFormat, "log-format", string(rootLog.DefaultFormat), "Set the log format")
	flag.StringVar(&KubeConfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")

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
	logger = log.NewLog()

	flag.Parse()
	SCCNamespace = os.Getenv("SCC_SYSTEM_NAMESPACE")
}

func main() {
	logger.Infof("Starting scc-operator version %s (%s) [built at %s]", version.Version, version.GitCommit, version.Date)
	ctx := signals.SetupSignalContext()
	restKubeConfig, err := kubeconfig.GetNonInteractiveClientConfig(KubeConfig).ClientConfig()
	if err != nil {
		logger.Fatalf("failed to find kubeconfig: %v", err)
	}

	dm := os.Getenv("CATTLE_DEV_MODE")
	util.SetDevMode(dm != "")
	runOptions := types.RunOptions{
		SccNamespace: SCCNamespace,
	}

	if err := operator.Run(ctx, restKubeConfig, runOptions); err != nil {
		logger.Fatalf("Error running operator: %s", err.Error())
	}
}
