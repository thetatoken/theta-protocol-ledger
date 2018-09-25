// +build unit

package util

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestParseLogLevelConfig(t *testing.T) {
	assert := assert.New(t)

	ret := parseLogLevelConfig("*:error,p2p:debug,consensus:info")
	assert.Equal(3, len(ret))
	assert.Equal("error", ret["*"])
	assert.Equal("debug", ret["p2p"])
	assert.Equal("info", ret["consensus"])

	// Should set default level.
	ret2 := parseLogLevelConfig("p2p:debug,consensus:info")
	assert.Equal(3, len(ret2))
	assert.Equal("warn", ret2["*"])
	assert.Equal("debug", ret2["p2p"])
	assert.Equal("info", ret2["consensus"])
}

func TestGetLoggerForModule(t *testing.T) {
	assert := assert.New(t)

	logLevels = parseLogLevelConfig("*:error,p2p:debug,consensus:info")

	assert.Equal(log.DebugLevel, GetLoggerForModule("p2p").Logger.Level)
	assert.Equal(log.InfoLevel, GetLoggerForModule("consensus").Logger.Level)
	assert.Equal(log.ErrorLevel, GetLoggerForModule("sync").Logger.Level)
}
