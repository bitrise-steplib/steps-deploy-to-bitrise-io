package uploaders

import (
	"github.com/bitrise-io/go-utils/log"
)

type logger struct{}

func NewLogger() *logger {
	return &logger{}
}

func (l *logger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (l *logger) Errorf(format string, v ...interface{}) {
	log.Errorf(format, v...)
}

func (l *logger) Warnf(format string, v ...interface{}) {
	log.Warnf(format, v...)
}

func (l *logger) AABParseWarnf(tag string, format string, v ...interface{}) {
	log.RWarnf("deploy-to-bitrise-io", tag, nil, format, v...)
}

func (l *logger) APKParseWarnf(tag string, format string, v ...interface{}) {
	log.RWarnf("deploy-to-bitrise-io", tag, nil, format, v...)
}
