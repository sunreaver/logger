package logger

import (
	"log/slog"
	"strings"
)

// LevelString 字符串格式的Level.
type LevelString string

const (
	// DebugStringLevel debug.
	DebugStringLevel LevelString = "debug"
	// InfoStringLevel info.
	InfoStringLevel LevelString = "info"
	// WarnStringLevel warn.
	WarnStringLevel LevelString = "warn"
	// ErrorStringLevel error.
	ErrorStringLevel LevelString = "error"
)

func (l LevelString) toLevel() slog.Level {
	switch LevelString(strings.ToLower(string(l))) {
	case DebugStringLevel:
		return slog.LevelDebug
	case InfoStringLevel:
		return slog.LevelInfo
	case WarnStringLevel:
		return slog.LevelWarn
	case ErrorStringLevel:
		return slog.LevelError
	}

	return slog.LevelDebug
}
