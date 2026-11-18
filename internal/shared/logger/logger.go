package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

func NewLogger(env string) *logrus.Logger {
	log := logrus.New()

	log.SetOutput(os.Stdout)
	configure(log, env)
	configure(logrus.StandardLogger(), env)

	return log
}

func configure(log *logrus.Logger, env string) {
	if env == "development" {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:          true,
			DisableLevelTruncation: true,
		})
	} else {
		log.SetFormatter(&logrus.JSONFormatter{})
	}

	if env == "production" {
		log.SetLevel(logrus.InfoLevel)
	} else {
		log.SetLevel(logrus.DebugLevel)
	}
}
