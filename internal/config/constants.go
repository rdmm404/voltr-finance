package config

import "log/slog"

type LogLevel string

const (
	LogLevelDebug   = "DEBUG"
	LogLevelInfo    = "INFO"
	LogLevelWarning = "WARNING"
	LogLevelError   = "ERROR"
)

func (l LogLevel) Valid() bool {
	switch l {
	case LogLevelDebug, LogLevelError, LogLevelInfo, LogLevelWarning:
		return true
	default:
		return false
	}
}

func (l LogLevel) ToSlog() slog.Level {
	return map[LogLevel]slog.Level{
		LogLevelDebug:   slog.LevelDebug,
		LogLevelError:   slog.LevelError,
		LogLevelInfo:    slog.LevelInfo,
		LogLevelWarning: slog.LevelWarn,
	}[l]
}
