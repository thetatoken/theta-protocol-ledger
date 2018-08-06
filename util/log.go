package util

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func init() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true

	if viper.GetBool(CfgLogDebug) {
		log.SetLevel(log.DebugLevel)
	}
}
