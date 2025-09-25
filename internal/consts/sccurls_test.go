//go:build test
// +build test

package consts

import (
	"testing"

	"github.com/rancher/scc-operator/internal/initializer"
	"github.com/stretchr/testify/assert"
)

func TestSCCEnvironment_String(t *testing.T) {
	asserts := assert.New(t)

	var tests = []struct {
		name  string
		input SCCEnvironment
		want  string
	}{
		{"Prod EnvKey", ProductionSCC, "production"},
		{"StagingSCC EnvKey", StagingSCC, "staging"},
		{"PAYG EnvKey", PayAsYouGo, "payAsYouGo"},
		{"RGS EnvKey", RGS, "rgs"},
		{"unknown", 42, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asserts.Equal(tt.want, tt.input.String())
		})
	}
}

func TestAlternativeSccURLs_Ptr(t *testing.T) {
	asserts := assert.New(t)

	var tests = []struct {
		name  string
		input AlternativeSccURLs
		want  string
	}{
		{"Prod URL", ProdSccURL, "https://scc.suse.com"},
		{"StagingSCC URL", StagingSccURL, "https://stgscc.suse.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asserts.Equal(tt.want, *tt.input.Ptr())
		})
	}
}

func TestGetSCCEnvironment_Dev(t *testing.T) {
	asserts := assert.New(t)
	initializer.DevMode.SetForTest(true)
	asserts.Equal(StagingSCC, GetSCCEnvironment())
}

func TestGetSCCEnvironment_Prod(t *testing.T) {
	asserts := assert.New(t)
	initializer.DevMode.SetForTest(false)
	asserts.Equal(ProductionSCC, GetSCCEnvironment())
}

func TestBaseURLForSCC_Dev(t *testing.T) {
	asserts := assert.New(t)
	initializer.DevMode.SetForTest(true)
	asserts.Equal(string(StagingSccURL), BaseURLForSCC())
}

func TestBaseURLForSCC_Prod(t *testing.T) {
	asserts := assert.New(t)
	initializer.DevMode.SetForTest(false)
	asserts.Equal(string(ProdSccURL), BaseURLForSCC())
}
