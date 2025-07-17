package settingrepo

import (
	"github.com/rancher-sandbox/scc-operator/pkg/util/log"
	"net/url"
)

const (
	SettingNameServerUrl   = "server-url"
	SettingNameInstallUUID = "install-uuid"
)

func GetServerURL(settings *SettingRepository) string {
	if settings == nil || !settings.HasSetting(SettingNameServerUrl) {
		return ""
	}
	serverUrlSetting, err := settings.GetSetting(SettingNameServerUrl)
	if err != nil {
		log.NewLog().Error(err, "Failed to get server url setting")
		return ""
	}

	return serverUrlSetting.Value
}

// ServerHostname returns the hostname of the Rancher server URL
func ServerHostname(settings *SettingRepository) string {
	serverUrl := GetServerURL(settings)
	if serverUrl == "" {
		return ""
	}
	parsed, _ := url.Parse(serverUrl)
	return parsed.Host
}

func GetRancherInstallUUID(settings *SettingRepository) string {
	if settings == nil || !settings.HasSetting(SettingNameInstallUUID) {
		return ""
	}

	installUUIDSetting, err := settings.GetSetting(SettingNameInstallUUID)
	if err != nil {
		log.NewLog().Error(err, "Failed to get install uuid setting")
		return ""
	}

	return installUUIDSetting.Value
}
