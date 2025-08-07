package rancher

import "net/url"

func GetServerURL() string {
	// TODO: maybe replace with helper to get server URL via other means?
	return ""
}

// ServerHostname returns the hostname of the Rancher server URL
func ServerHostname() string {
	serverURL := GetServerURL()
	if serverURL == "" {
		return ""
	}
	parsed, _ := url.Parse(serverURL)
	return parsed.Host
}

func GetRancherInstallUUID() string {
	// TODO: maybe replace with helper to get UUID via other means?
	return ""
}
