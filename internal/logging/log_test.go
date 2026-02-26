package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLog(t *testing.T) {
	assert.NotNil(t, rootLogger)
}
