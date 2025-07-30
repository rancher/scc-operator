package controllers

import (
	"time"

	"github.com/rancher/scc-operator/internal/initializer"
)

func minResyncInterval() time.Time {
	now := time.Now()
	if initializer.DevMode.Get() {
		return now.Add(-devMinCheckin)
	}
	return now.Add(-prodMinCheckin)
}
