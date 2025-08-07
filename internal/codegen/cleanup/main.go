package main

import (
	"os"

	"github.com/rancher/wrangler/v3/pkg/cleanup"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	if err := os.RemoveAll("./pkg/client/generated"); err != nil {
		return err
	}
	if err := os.RemoveAll("./pkg/crds/yamls/generated"); err != nil {
		return err
	}
	if err := os.RemoveAll("./pkg/generated"); err != nil {
		return err
	}

	if err := os.RemoveAll("./internal/rancher/client/generated"); err != nil {
		return err
	}
	if err := os.RemoveAll("./internal/rancher/crds/yamls/generated"); err != nil {
		return err
	}
	if err := os.RemoveAll("./internal/rancher/generated"); err != nil {
		return err
	}

	if err := cleanup.Cleanup("./internal/rancher/apis"); err != nil {
		return err
	}

	return cleanup.Cleanup("./pkg/apis")
}
