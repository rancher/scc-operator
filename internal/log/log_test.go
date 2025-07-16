package log

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewLog(t *testing.T) {
	assert.NotNil(t, rootLogger)
}
