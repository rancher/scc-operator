package consts

import "github.com/rancher/scc-operator/internal/initializer"

type SCCEnvironment int

const (
	Production SCCEnvironment = iota
	Staging
	PayAsYouGo
	RGS
)

func (s SCCEnvironment) String() string {
	switch s {
	case Production:
		return "production"
	case Staging:
		return "staging"
	case PayAsYouGo:
		return "payAsYouGo"
	case RGS:
		return "rgs"
	default:
		return "unknown"
	}
}

func GetSCCEnvironment() SCCEnvironment {
	if !initializer.DevMode.Get() {
		return Production
	}
	return Staging
}

type AlternativeSccURLs string

const (
	ProdSccURL    AlternativeSccURLs = "https://scc.suse.com"
	StagingSccURL AlternativeSccURLs = "https://stgscc.suse.com"
)

// TODO in the future we can store the PAYG and other urls too

func (s AlternativeSccURLs) Ptr() *string {
	stringVal := string(s)
	return &stringVal
}

func BaseURLForSCC() string {
	var baseURL string
	switch GetSCCEnvironment() {
	case Production:
		baseURL = string(ProdSccURL)
	case Staging:
		baseURL = string(StagingSccURL)
	case RGS: // explicitly return empty for RGS
	default:
		// intentionally do nothing and return empty string
	}

	return baseURL
}
