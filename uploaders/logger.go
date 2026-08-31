package uploaders

import (
	"github.com/bitrise-io/go-utils/v2/log"
)

type metaparserLogger struct {
	logger log.Logger
}

func NewLogger(logger log.Logger) *metaparserLogger {
	return &metaparserLogger{logger: logger}
}

func (l *metaparserLogger) Warnf(format string, v ...interface{}) {
	l.logger.Warnf(format, v...)
}

// AABParseWarnf and APKParseWarnf are telemetry hooks the metaparser calls on
// parse failures. The step does not collect this telemetry, and the failures
// are already surfaced to the user via Warnf, so these are intentionally no-ops.
func (l *metaparserLogger) AABParseWarnf(_ string, _ string, _ ...interface{}) {}

func (l *metaparserLogger) APKParseWarnf(_ string, _ string, _ ...interface{}) {}
