package util

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/common"
)

var logLevels map[string]string

const (
	panicLevel = "panic"
	fatalLevel = "fatal"
	errorLevel = "error"
	warnLevel  = "warn"
	infoLevel  = "info"
	debugLevel = "debug"
)
const defaultLevel = warnLevel

func init() {
	logLevels = parseLogLevelConfig(viper.GetString(common.CfgLogLevels))
}

func parseLogLevelConfig(config string) map[string]string {
	levels := make(map[string]string)

	moduleAndLevels := strings.Split(config, ",")
	for _, moduleAndLevel := range moduleAndLevels {
		tokens := strings.Split(moduleAndLevel, ":")
		if len(tokens) != 2 {
			panic(fmt.Sprintf("Failed to parse module log level: \"%v\"", moduleAndLevel))
		}
		levels[strings.TrimSpace(tokens[0])] = strings.TrimSpace(tokens[1])
	}

	if _, ok := levels["*"]; !ok {
		levels["*"] = defaultLevel
	}
	return levels
}

// GetLoggerForModule returns the logger for given module.
func GetLoggerForModule(module string) *log.Entry {
	customFormatter := new(TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	customFormatter.ForceFormatting = true

	logger := log.New()
	logger.Formatter = customFormatter

	level, ok := logLevels[module]
	if !ok {
		level = logLevels["*"]
	}

	if level == panicLevel {
		logger.SetLevel(log.PanicLevel)
	} else if level == fatalLevel {
		logger.SetLevel(log.FatalLevel)
	} else if level == errorLevel {
		logger.SetLevel(log.ErrorLevel)
	} else if level == warnLevel {
		logger.SetLevel(log.WarnLevel)
	} else if level == infoLevel {
		logger.SetLevel(log.InfoLevel)
	} else if level == debugLevel {
		logger.SetLevel(log.DebugLevel)
	}

	return logger.WithFields(log.Fields{"prefix": module})
}
