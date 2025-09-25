package consts

import "github.com/rancher/scc-operator/internal/initializer"

type SCCEnvironment int

const (
	ProductionSCC SCCEnvironment = iota
	StagingSCC
	PayAsYouGo //Disabled for now
	RGS        // Shouldn't matter for now until RGS supported
)

func (s SCCEnvironment) String() string {
	switch s {
	case ProductionSCC:
		return "production"
	case StagingSCC:
		return "staging"
	case PayAsYouGo:
		return "payAsYouGo"
	case RGS:
		return "rgs"
	default:
		return "unknown"
	}
}

// BaseURLForSCC returns an environment's default SCC URL (or empty string if SCC not used)
func (s SCCEnvironment) BaseURLForSCC() string {
	var baseURL string
	switch s {
	case ProductionSCC:
		baseURL = string(ProdSccURL)
	case StagingSCC:
		baseURL = string(StagingSccURL)
	case RGS, PayAsYouGo:
		fallthrough
	default:
		// intentionally do nothing and return empty string
	}

	return baseURL
}

func GetSCCEnvironment() SCCEnvironment {
	if !initializer.DevMode.Get() {
		return ProductionSCC
	}
	return StagingSCC
}

type SccURLs string

const (
	ProdSccURL    SccURLs = "https://scc.suse.com"
	StagingSccURL SccURLs = "https://stgscc.suse.com"
)

func (s SccURLs) Ptr() *string {
	stringVal := string(s)
	return &stringVal
}

// BaseURLForSCC returns the SCC URL (or empty string) for the detected environment
func BaseURLForSCC() string {
	return GetSCCEnvironment().BaseURLForSCC()
}
