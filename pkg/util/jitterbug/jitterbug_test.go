package jitterbug

import (
	"math/rand"
	"testing"
	"time"

	"github.com/rancher/scc-operator/pkg/util/log"
	"github.com/stretchr/testify/assert"
)

var testLogs = log.NewComponentLogger("jitterbug-tests")

func TestNewJitterChecker(t *testing.T) {
	t.Parallel()
	config := &Config{
		BaseInterval: 10 * time.Hour,
	}

	lastCheck := time.Now()
	jitterChecker := NewJitterChecker(
		config,
		func(dailyTriggerTime, maxTriggerTime time.Duration) (bool, error) {
			timeDiff := time.Since(lastCheck)
			if timeDiff > dailyTriggerTime {
				testLogs.Infof("Hi IAN!")
				lastCheck = time.Now()
				return true, nil
			}
			return false, nil
		},
	)

	assert.Equal(t, jitterChecker.config, config)
}

func TestNewJitterCheckerFromCalculator(t *testing.T) {
	t.Parallel()
	config := &Config{
		BaseInterval: 10 * time.Hour,
	}

	// Use a deterministic random source
	seed := int64(42)
	r := rand.New(rand.NewSource(seed))
	lastCheck := time.Now()

	jitterCacl := NewJitterCalculator(config, r)
	jitterChecker := NewJitterCheckerFromCalculator(
		*jitterCacl,
		func(dailyTriggerTime, maxTriggerTime time.Duration) (bool, error) {
			timeDiff := time.Since(lastCheck)
			if timeDiff > dailyTriggerTime {
				testLogs.Infof("Hi IAN!")
				lastCheck = time.Now()
				return true, nil
			}
			return false, nil
		},
	)

	assert.Equal(t, jitterChecker.config, config)
}
