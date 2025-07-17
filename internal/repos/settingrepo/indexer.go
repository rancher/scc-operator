package settingrepo

import (
	mgmtv3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
)

const (
	IndexSettingByName = "scc.io/setting-by-name"
)

func (repo *SettingRepository) InitIndexers() {
	repo.Cache.AddIndexer(
		IndexSettingByName,
		func(setting *mgmtv3.Setting) ([]string, error) {
			if setting == nil {
				return nil, nil
			}

			if setting.Name != SettingNameInstallUUID && setting.Name != SettingNameServerUrl {
				return nil, nil
			}
			return []string{setting.Name}, nil
		},
	)
}
