package crds

// RequiredCRDs returns a list of CRD to install based on enabled features.
func RequiredCRDs() []string {
	return []string{
		"registrations.scc.cattle.io",
	}
}
