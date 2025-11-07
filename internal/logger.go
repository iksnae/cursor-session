package internal

import (
	"log"
	"os"
)

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelError LogLevel = iota
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

var (
	logLevel = LogLevelInfo
	logger   = log.New(os.Stderr, "", log.LstdFlags)
)

// SetLogLevel sets the global log level
func SetLogLevel(level LogLevel) {
	logLevel = level
}

// SetVerbose enables verbose (debug) logging
func SetVerbose(verbose bool) {
	if verbose {
		SetLogLevel(LogLevelDebug)
	} else {
		SetLogLevel(LogLevelInfo)
	}
}

func logError(format string, args ...interface{}) {
	if logLevel >= LogLevelError {
		logger.Printf("[ERROR] "+format, args...)
	}
}

func logWarn(format string, args ...interface{}) {
	if logLevel >= LogLevelWarn {
		logger.Printf("[WARN] "+format, args...)
	}
}

func logInfo(format string, args ...interface{}) {
	if logLevel >= LogLevelInfo {
		logger.Printf("[INFO] "+format, args...)
	}
}

func logDebug(format string, args ...interface{}) {
	if logLevel >= LogLevelDebug {
		logger.Printf("[DEBUG] "+format, args...)
	}
}

// LogError logs an error message
func LogError(format string, args ...interface{}) {
	logError(format, args...)
}

// LogWarn logs a warning message
func LogWarn(format string, args ...interface{}) {
	logWarn(format, args...)
}

// LogInfo logs an info message
func LogInfo(format string, args ...interface{}) {
	logInfo(format, args...)
}

// LogDebug logs a debug message
func LogDebug(format string, args ...interface{}) {
	logDebug(format, args...)
}
