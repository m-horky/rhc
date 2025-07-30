package logging

import (
	"log/slog"
)

// LogMessage represents a message with an associated log level
type LogMessage struct {
	Level   slog.Level
	Message error
}

// HasPriorityErrors checks if the errorMessage map has any error
// with a higher priority than the logLevel configure.
func HasPriorityErrors(errorMessages map[string]LogMessage, level slog.Level) bool {
	for _, logMsg := range errorMessages {
		if logMsg.Level >= level {
			return true
		}
	}
	return false
}