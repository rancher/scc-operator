package logging

import (
	"github.com/sirupsen/logrus"
)

type StructuredLogger = *logrus.Entry

type StructuredLoggerBuilder struct {
	devMode           bool
	Component         string
	Controller        *string
	SubComponent      *string
	OperatorName      *string
	OperatorNamespace *string
}

type Optional func(*StructuredLoggerBuilder)

func NewStructuredLoggerBuilder(component string, optional ...Optional) *StructuredLoggerBuilder {
	logger := StructuredLoggerBuilder{
		Component: component,
	}

	logger.devMode = false
	//if consts.IsDevMode() {
	//	logger.devMode = true
	//}

	for _, opt := range optional {
		opt(&logger)
	}

	return &logger
}

func WithOperatorName(operatorName string) Optional {
	return func(logger *StructuredLoggerBuilder) {
		logger.OperatorName = &operatorName
	}
}

func WithOperatorNamespace(operatorNamespace string) Optional {
	return func(logger *StructuredLoggerBuilder) {
		logger.OperatorNamespace = &operatorNamespace
	}
}

func WithController(controller string) Optional {
	return func(logger *StructuredLoggerBuilder) {
		logger.Controller = &controller
	}
}

func WithSubComponent(subComponent string) Optional {
	return func(logger *StructuredLoggerBuilder) {
		logger.SubComponent = &subComponent
	}
}

func newStructuredLog(component string) StructuredLogger {
	if rootLogger == nil {
		rootLogger = logrus.StandardLogger()
	}

	baseLogEntry := rootLogger.
		WithField("component", component)

	return baseLogEntry
}

func NewStructuredLogger(component string, optional ...Optional) StructuredLogger {
	builder := NewStructuredLoggerBuilder(component, optional...)

	return builder.ToLogger()
}

func (lb *StructuredLoggerBuilder) ToLogger() StructuredLogger {
	baseLogEntry := newStructuredLog(lb.Component)

	if lb.devMode {
		baseLogEntry = baseLogEntry.WithField("devMode", lb.devMode)
	}

	if lb.Controller != nil && *lb.Controller != "" {
		baseLogEntry = baseLogEntry.WithField("controller", *lb.Controller)
	}

	if lb.SubComponent != nil && *lb.SubComponent != "" {
		baseLogEntry = baseLogEntry.WithField("subcomponent", *lb.SubComponent)
	}

	return baseLogEntry
}
