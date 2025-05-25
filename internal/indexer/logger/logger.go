package logger

import (
	"io"
	"log"
	"os"
	"strings"
)

// LogLevel type for defining logging levels
type LogLevel int

const (
	// DEBUG logs everything
	DEBUG LogLevel = iota
	// INFO logs informational messages (default)
	INFO
	// WARNING logs warnings and errors
	WARNING
	// ERROR logs only errors
	ERROR
)

var currentLevel = INFO // Default log level

// stringToLogLevel converts a string to a LogLevel const
func stringToLogLevel(levelStr string) LogLevel {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARNING":
		return WARNING
	case "ERROR":
		return ERROR
	default:
		log.Printf("Warning: Unknown log level '%s'. Defaulting to INFO.", levelStr)
		return INFO
	}
}

// Init initializes the logger with a given level string and output writer.
// If out is nil, os.Stderr is used.
func Init(levelStr string, out io.Writer) {
	currentLevel = stringToLogLevel(levelStr)
	if out == nil {
		out = os.Stderr
	}
	log.SetOutput(out)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile) // Include file/line for debug context
	log.Printf("Logger initialized with level: %s", levelStr)
}

// Debugf logs a formatted debug message if current level allows.
func Debugf(format string, v ...interface{}) {
	if currentLevel <= DEBUG {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// Infof logs a formatted info message if current level allows.
func Infof(format string, v ...interface{}) {
	if currentLevel <= INFO {
		log.Printf("[INFO] "+format, v...)
	}
}

// Warnf logs a formatted warning message if current level allows.
func Warnf(format string, v ...interface{}) {
	if currentLevel <= WARNING {
		log.Printf("[WARN] "+format, v...)
	}
}

// Errorf logs a formatted error message if current level allows.
// It does not exit.
func Errorf(format string, v ...interface{}) {
	if currentLevel <= ERROR {
		log.Printf("[ERROR] "+format, v...)
	}
}

// Fatalf logs a formatted error message and then calls os.Exit(1).
// It always logs, regardless of level, as it's a fatal error.
func Fatalf(format string, v ...interface{}) {
	log.Fatalf("[FATAL] "+format, v...) // Standard log.Fatalf includes file/line and exits
}

// Printf uses the standard log.Printf, typically for messages that should always appear or for fallback.
// It will respect the log.SetOutput and log.SetFlags.
// Consider using level-specific functions for most logging.
func Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}
