package rancher

import (
	"context"

	"github.com/rancher/scc-operator/internal/consts"
	"github.com/rancher/scc-operator/internal/rancher/settings"
	"github.com/rancher/scc-operator/pkg/util/log"
	"github.com/sirupsen/logrus"
)

func GetServerURL(ctx context.Context, settings *settings.SettingReader) string {
	if settings == nil || !settings.Has(ctx, consts.SettingNameServerURL) {
		return ""
	}

	serverUrlSetting, err := settings.Get(ctx, consts.SettingNameServerURL)
	if err != nil {
		log.NewLog().Error(err, "Failed to get install uuid setting")
		return ""
	}
	logrus.Debug(serverUrlSetting)

	return serverUrlSetting.Get()
}

func GetRancherInstallUUID(ctx context.Context, settings *settings.SettingReader) string {
	if settings == nil || !settings.Has(ctx, consts.SettingNameInstallUUID) {
		return ""
	}

	installUUIDSetting, err := settings.Get(ctx, consts.SettingNameInstallUUID)
	if err != nil {
		log.NewLog().Error(err, "Failed to get install uuid setting")
		return ""
	}
	logrus.Debug(installUUIDSetting)

	return installUUIDSetting.Get()
}
