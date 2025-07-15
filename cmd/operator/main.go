package main

import (
	"flag"
	"github.com/sirupsen/logrus"

	"github.com/rancher-sandbox/scc-operator/cmd/operator/version"
	rootLog "github.com/rancher-sandbox/scc-operator/internal/log"
	"github.com/rancher-sandbox/scc-operator/pkg/log"
)

const (
	LogDateFormat = "2006/01/02 15:04:05"
)

var (
	LogFormat       string
	KubeConfig      string
	SystemNamespace string
	Debug           bool
	Trace           bool
	logger          rootLog.StructuredLogger
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
}

func main() {
	logger.Infof("Starting scc-operator version %s (%s) [built at %s]", version.Version, version.GitCommit, version.Date)
}
