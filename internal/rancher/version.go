package rancher

import (
	"regexp"
	"strings"
)

// semverRegex matches on regular SemVer as well as Rancher's dev versions
var semverRegex = regexp.MustCompile(`(?m)^v?(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)(?:\.(?P<patch>0|[1-9]\d*))?(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

type Version string

// versionIsDevBuild this should only ever be used for SCC systeminfo module
func (rv Version) versionIsDevBuild() bool {
	if rv == "dev" {
		return true
	}

	matches := semverRegex.FindStringSubmatch(string(rv))
	return matches == nil || // When version is not SemVer it is dev
		matches[3] == "" || // When the version is just Major.Minor assume dev
		matches[4] != "" // When the version includes pre-release assume dev
}

// SCCSafeVersion returns the version to be used when submitting product registration info to SCC
// Notably this is necessary for product information specifically, other metrics may report "true" rancher version if allowed
func (rv Version) SCCSafeVersion() string {
	if rv.versionIsDevBuild() {
		return "other"
	}
	if strings.HasPrefix(string(rv), "v") {
		return strings.TrimPrefix(string(rv), "v")
	}
	return string(rv)
}
