package main

import (
	"flag"

	//"github.com/rancher/wrangler/v3/pkg/kubeconfig"
	//"github.com/rancher/wrangler/v3/pkg/signals"
	"github.com/sirupsen/logrus"

	"github.com/rancher-sandbox/scc-operator/cmd/operator/version"
)

const (
	LogFormat = "2006/01/02 15:04:05"
)

var (
	KubeConfig      string
	SystemNamespace string
	Debug           bool
	Trace           bool
)

func init() {
	flag.StringVar(&KubeConfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.BoolVar(&Debug, "debug", false, "Enable debug logging.")
	flag.BoolVar(&Trace, "trace", false, "Enable trace logging.")

	flag.Parse()
}

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true, ForceColors: true, TimestampFormat: LogFormat})
	if Debug {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debugf("Loglevel set to [%v]", logrus.DebugLevel)
	}
	if Trace {
		logrus.SetLevel(logrus.TraceLevel)
		logrus.Tracef("Loglevel set to [%v]", logrus.TraceLevel)
	}

	logrus.Infof("Starting scc-operator version %s (%s) [built at %s]", version.Version, version.GitCommit, version.Date)
}
