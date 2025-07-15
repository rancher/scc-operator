package util

import "context"

var (
	systemNamespace string
	initNs          = Initializer{}
	devMode         bool
	initDev         = Initializer{}
)

func SetDevMode(enabled bool) {
	initDev.InitOnce(func() {
		devMode = enabled
	})
}

func DevMode() bool {
	initDev.WaitForInit()
	return devMode
}

func DevModeContext(ctx context.Context) bool {
	err := initDev.WaitForInitContext(ctx)
	if err != nil {
		return false
	}
	return devMode
}

func GetSystemNamespaceContext(ctx context.Context) (string, error) {
	if err := initNs.WaitForInitContext(ctx); err != nil {
		return "", err
	}
	return systemNamespace, nil
}

func GetSystemNamespace() string {
	initNs.WaitForInit()
	return systemNamespace
}

func SetSystemNamespace(ns string) {
	initNs.InitOnce(func() {
		systemNamespace = ns
	})
}
