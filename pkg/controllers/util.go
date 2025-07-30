package controllers

import (
	"time"

	"github.com/rancher/scc-operator/internal/util"
)

func minResyncInterval() time.Time {
	now := time.Now()
	if util.DevMode.Get() {
		return now.Add(-devMinCheckin)
	}
	return now.Add(-prodMinCheckin)
}
