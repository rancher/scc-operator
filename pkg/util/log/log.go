package log

import "github.com/rancher/scc-operator/internal/log"

var defaultOpts []log.Optional

func AddDefaultOpts(opts ...log.Optional) {
	defaultOpts = append(defaultOpts, opts...)
}

func NewSccLogBuilder(opts ...log.Optional) *log.StructuredLoggerBuilder {
	logOpts := append(defaultOpts, opts...)
	return log.NewStructuredLoggerBuilder("scc-operator", logOpts...)
}

func NewLog(opts ...log.Optional) log.StructuredLogger {
	baseLogger := NewSccLogBuilder(opts...).ToLogger()

	return baseLogger
}

func NewControllerLogger(controllerName string) log.StructuredLogger {
	builder := NewSccLogBuilder(log.WithController(controllerName))

	return builder.ToLogger()
}
