package stargate

type loggerFunc func(format string, a ...any)

var Logger loggerFunc

func v(format string, a ...any) {
	if Logger != nil {
		Logger(format, a...)
	}
}
