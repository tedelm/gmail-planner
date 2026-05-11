package logging

import (
	"log"
)

// Logger wraps the standard log.Logger with additional functionality
type Logger struct {
	*log.Logger
}

// LogLevel represents different logging levels
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// LogConfig holds configuration for logging
type LogConfig struct {
	Level  LogLevel
	Prefix string
	Flags  int
}

// DefaultLogConfig returns a default logging configuration
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:  INFO,
		Prefix: "[APP] ",
		Flags:  log.LstdFlags | log.Lshortfile,
	}
}
