package settingrepo

import (
	"net/url"

	"github.com/rancher/scc-operator/pkg/util/log"
)

const (
	SettingNameServerURL   = "server-url"
	SettingNameInstallUUID = "install-uuid"
)

func GetServerURL(settings *SettingRepository) string {
	if settings == nil || !settings.HasSetting(SettingNameServerURL) {
		return ""
	}
	serverURLSetting, err := settings.GetSetting(SettingNameServerURL)
	if err != nil {
		log.NewLog().Error(err, "Failed to get server url setting")
		return ""
	}

	return serverURLSetting.Value
}

// ServerHostname returns the hostname of the Rancher server URL
func ServerHostname(settings *SettingRepository) string {
	serverURL := GetServerURL(settings)
	if serverURL == "" {
		return ""
	}
	parsed, _ := url.Parse(serverURL)
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
