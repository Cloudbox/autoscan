package bernard

type noLogger struct{}

func newNoLogger() *noLogger {
	return &noLogger{}
}

func (noLogger) Info(msg string, keysAndValues ...interface{}) {}

func (noLogger) Error(err error, msg string, keysAndValues ...interface{}) {}
