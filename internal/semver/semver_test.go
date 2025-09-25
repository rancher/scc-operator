package semver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRancherVersion(t *testing.T) {
	asserts := assert.New(t)

	var tests = []struct {
		name       string
		input      string
		isDevBuild bool
		sccVersion string
	}{
		{
			"Dev IDE",
			"dev",
			true,
			"other",
		},
		{
			"Dev Build/Head Images",
			"v2.12-207d1eaa2-head",
			true,
			"other",
		},
		{
			"Alpha Build",
			"v2.12.1-alpha4",
			true,
			"other",
		},
		{
			"Release",
			"v2.12.1",
			false,
			"2.12.1",
		},
		{
			"Manual Override",
			"2.13.1",
			false,
			"2.13.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testVersion := Version(tt.input)
			asserts.Equal(tt.isDevBuild, testVersion.versionIsDevBuild())
			asserts.Equal(tt.sccVersion, testVersion.SCCSafeVersion())
		})
	}

}
