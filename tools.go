//go:generate go run internal/codegen/cleanup/main.go
//go:generate go run internal/codegen/main.go
//go:generate go run internal/codegen/crds/main.go
package main

import (
	_ "sigs.k8s.io/controller-tools/pkg/version"
)
