package androidartifact

type Logger interface {
	Warnf(format string, args ...interface{})
	AABParseWarnf(tag string, format string, v ...interface{})
	APKParseWarnf(tag string, format string, v ...interface{})
}
