package log

import (
	"github.com/ehazlett/simplelog"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
	"time"
)

type Format string

// IsValid checks if the LogFormat is one of the defined constants.
func (lf Format) IsValid() bool {
	switch lf {
	case FormatSimple, FormatText, FormatJSON:
		return true
	}
	return false
}

const (
	FormatSimple  Format = "simple"
	FormatText    Format = "text"
	FormatJSON    Format = "json"
	DefaultFormat        = FormatText
)

var (
	// Logger is the globally accessible logger instance.
	rootLogger       *logrus.Logger
	currentLogLevel  logrus.Level // Store the actual logrus.Level
	currentLogFormat Format       // To store the string representation of the format
)

func init() {
	rootLogger = logrus.StandardLogger()
	rootLogger.SetOutput(os.Stdout) // Default output to stdout

	// Set initial defaults
	SetLogLevel(logrus.InfoLevel)
	SetLogFormat(FormatText) // Default to text format
}

// SetLogLevel sets the logging level for the global logger using logrus.Level.
func SetLogLevel(level logrus.Level) {
	currentLogLevel = level
	rootLogger.SetLevel(level)
	rootLogger.Debugf("Log level set to: %s", level.String())
}

// ParseAndSetLogLevelFromString parses a string and sets the logging level.
func ParseAndSetLogLevelFromString(levelStr string) {
	parsedLevel, err := logrus.ParseLevel(strings.ToLower(levelStr))
	if err != nil {
		rootLogger.Warnf("Invalid log level string '%s' provided. Defaulting to 'info'. Error: %v", levelStr, err)
		SetLogLevel(logrus.InfoLevel)
	} else {
		SetLogLevel(parsedLevel)
	}
}

// GetLogLevel returns the currently set log level as a logrus.Level.
func GetLogLevel() logrus.Level {
	return currentLogLevel
}

// SetLogFormat sets the logging output format for the global logger
func SetLogFormat(format Format) {
	currentLogFormat = format
	switch currentLogFormat {
	case FormatJSON:
		rootLogger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
		rootLogger.Debugf("Log format set to: %s", FormatJSON)
	case FormatSimple:
		rootLogger.Warnf("Invalid log format '%s' provided. Defaulting to '%s'.", format, DefaultFormat)
		rootLogger.SetFormatter(&simplelog.StandardFormatter{})
		currentLogFormat = DefaultFormat
	case FormatText:
	default:
		rootLogger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
		rootLogger.Debugf("Log format set to: %s", FormatText)
	}
}

// GetLogFormat returns the currently set log format
func GetLogFormat() Format {
	return currentLogFormat
}

// ParseAndSetLogFormatFromString parses a string and sets the logging format
func ParseAndSetLogFormatFromString(formatStr string) {
	logFormat := Format(strings.ToLower(formatStr))
	if !logFormat.IsValid() {
		rootLogger.Warnf("Invalid log format '%s' provided. Defaulting to '%s'.", formatStr, DefaultFormat)
		logFormat = DefaultFormat
	}

	SetLogFormat(logFormat) // Directly call SetLogFormat as it already handles validation
}

func SetLevelFromEnvironment() {
	if os.Getenv("CATTLE_TRACE") == "true" || os.Getenv("RANCHER_TRACE") == "true" {
		SetLogLevel(logrus.TraceLevel)
	} else if os.Getenv("CATTLE_DEBUG") == "true" || os.Getenv("RANCHER_DEBUG") == "true" {
		SetLogLevel(logrus.DebugLevel)
	}
}
