package controllers

import (
	"github.com/rancher/scc-operator/internal/util"
	"time"
)

func minResyncInterval() time.Time {
	now := time.Now()
	if util.DevMode() {
		return now.Add(-devMinCheckin)
	}
	return now.Add(-prodMinCheckin)
}
