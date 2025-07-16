package settings

import (
	v3ctrl "github.com/rancher-sandbox/scc-operator/pkg/generated/controllers/management.cattle.io/v3"
	v3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var rootSettingRepo *SettingRepo

type SettingRepo struct {
	settings      v3ctrl.SettingController
	settingsCache v3ctrl.SettingCache
}

func NewSettingRepository(
	settings v3ctrl.SettingController,
	settingsCache v3ctrl.SettingCache,
) *SettingRepo {
	if rootSettingRepo == nil {
		rootSettingRepo = &SettingRepo{
			settings:      settings,
			settingsCache: settingsCache,
		}
		rootSettingRepo.initIndexers()
	}
	return rootSettingRepo
}

func (repo *SettingRepo) HasSetting(name string) bool {
	_, err := repo.settings.Get(name, metav1.GetOptions{})
	return err == nil
}

func (repo *SettingRepo) GetSetting(name string) (*v3.Setting, error) {
	return repo.settings.Get(name, metav1.GetOptions{})
}
