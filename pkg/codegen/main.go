// This program generates the code for the Rancher types and clients.
package main

import (
	"os"

	mgmtv3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	controllergen "github.com/rancher/wrangler/v3/pkg/controller-gen"
	"github.com/rancher/wrangler/v3/pkg/controller-gen/args"
)

func main() {
	_ = os.Unsetenv("GOPATH")

	controllergen.Run(args.Options{
		OutputPackage: "github.com/rancher/scc-operator/pkg/generated",
		Boilerplate:   "scripts/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"management.cattle.io": {
				PackageName: "management.cattle.io",
				Types: []interface{}{
					// All structs with an embedded ObjectMeta field will be picked up
					mgmtv3.Setting{},
					mgmtv3.Cluster{},
					mgmtv3.Node{},
				},
			},
			"scc.cattle.io": {
				PackageName: "scc.cattle.io",
				Types: []interface{}{
					// All structs with an embedded ObjectMeta field will be picked up
					"./pkg/apis/scc.cattle.io/v1",
				},
				GenerateTypes:   true,
				GenerateClients: true,
				GenerateOpenAPI: true,
				OpenAPIDependencies: []string{
					"k8s.io/apimachinery/pkg/apis/meta/v1",
				},
			},
		},
	})
}
