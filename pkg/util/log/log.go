package log

import "github.com/rancher-sandbox/scc-operator/internal/log"

func NewSccLogBuilder(opts ...log.Optional) *log.StructuredLoggerBuilder {
	return log.NewStructuredLoggerBuilder("scc-operator", opts...)
}

func NewLog() log.StructuredLogger {
	baseLogger := NewSccLogBuilder().ToLogger()

	return baseLogger
}

func NewControllerLogger(controllerName string) log.StructuredLogger {
	builder := NewSccLogBuilder(log.WithController(controllerName))

	return builder.ToLogger()
}
