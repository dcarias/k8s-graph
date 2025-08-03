package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// LogLevel represents the logging level
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var (
	currentLevel LogLevel = INFO
	logger       *log.Logger
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel converts a string to LogLevel
func ParseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}

// Init initializes the logger with the specified level
func Init(level LogLevel) {
	currentLevel = level
	logger = log.New(os.Stdout, "", log.LstdFlags)
}

// InitFromEnv initializes the logger from environment variable
func InitFromEnv() {
	levelStr := os.Getenv("KUBEGRAPH_LOG_LEVEL")
	if levelStr == "" {
		levelStr = "INFO" // default level
	}
	Init(ParseLogLevel(levelStr))
}

// shouldLog checks if the given level should be logged
func shouldLog(level LogLevel) bool {
	return level >= currentLevel
}

// logf formats and logs a message if the level is enabled
func logf(level LogLevel, format string, args ...interface{}) {
	if shouldLog(level) {
		message := fmt.Sprintf(format, args...)
		logger.Printf("[%s] %s", level.String(), message)
	}
}

// Debug logs a debug message
func Debug(format string, args ...interface{}) {
	logf(DEBUG, format, args...)
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	logf(INFO, format, args...)
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	logf(WARN, format, args...)
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	logf(ERROR, format, args...)
}

// GetLevel returns the current log level
func GetLevel() LogLevel {
	return currentLevel
}

// IsDebugEnabled returns true if debug logging is enabled
func IsDebugEnabled() bool {
	return shouldLog(DEBUG)
}
