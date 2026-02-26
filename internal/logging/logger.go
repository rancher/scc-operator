package logging

// Logger references a global logger instance great for one-off logging.
// Sometimes it doesn't make sense to create a component specific logger so we can use this one.
var Logger = NewLog(WithSubComponent("global-logger"))

var defaultOpts []Optional

func AddDefaultOpts(opts ...Optional) {
	defaultOpts = append(defaultOpts, opts...)
}

func NewSccLogBuilder(opts ...Optional) *StructuredLoggerBuilder {
	logOpts := append(defaultOpts, opts...)
	return NewStructuredLoggerBuilder("scc-operator", logOpts...)
}

func NewLog(opts ...Optional) StructuredLogger {
	baseLogger := NewSccLogBuilder(opts...).ToLogger()

	return baseLogger
}

func NewControllerLogger(controllerName string) StructuredLogger {
	builder := NewSccLogBuilder(WithController(controllerName))

	return builder.ToLogger()
}

func NewComponentLogger(componentName string) StructuredLogger {
	builder := NewSccLogBuilder(WithSubComponent(componentName))

	return builder.ToLogger()
}
