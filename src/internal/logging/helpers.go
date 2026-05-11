package logging

import (
	"io"
	"log"
	"os"
)

// NewLogger creates a new logger with the given configuration
func NewLogger(config *LogConfig) *Logger {
	if config == nil {
		config = DefaultLogConfig()
	}

	// Set output to stdout by default
	output := io.Writer(os.Stdout)

	// Create logger with configuration
	logger := log.New(output, config.Prefix, config.Flags)

	return &Logger{Logger: logger}
}

// NewFileLogger creates a new logger that writes to a file
func NewFileLogger(filename string, config *LogConfig) (*Logger, error) {
	if config == nil {
		config = DefaultLogConfig()
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	logger := log.New(file, config.Prefix, config.Flags)
	return &Logger{Logger: logger}, nil
}

// Debug logs a debug message
func (l *Logger) Debug(v ...interface{}) {
	l.Print("[DEBUG] ", v)
}

// Info logs an info message
func (l *Logger) Info(v ...interface{}) {
	l.Print("[INFO] ", v)
}

// Warn logs a warning message
func (l *Logger) Warn(v ...interface{}) {
	l.Print("[WARN] ", v)
}

// Error logs an error message
func (l *Logger) Error(v ...interface{}) {
	l.Print("[ERROR] ", v)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, v ...interface{}) {
	l.Printf("[DEBUG] "+format, v...)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, v ...interface{}) {
	l.Printf("[INFO] "+format, v...)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, v ...interface{}) {
	l.Printf("[WARN] "+format, v...)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.Printf("[ERROR] "+format, v...)
}
