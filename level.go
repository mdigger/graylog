package graylog

import (
	"golang.org/x/exp/slog"
)

// priority is a Graylog level.
type priority uint8

const (
	// From /usr/include/sys/syslog.h.
	log_EMERG   priority = 0
	log_ALERT   priority = 1
	log_CRIT    priority = 2
	log_ERR     priority = 3
	log_WARNING priority = 4
	log_NOTICE  priority = 5
	log_INFO    priority = 6
	log_DEBUG   priority = 7
)

func level(l slog.Level) priority {
	switch {
	case l < slog.LevelDebug:
		return log_DEBUG + priority(slog.LevelDebug-l.Level())
	case l < slog.LevelInfo:
		return log_DEBUG
	case l == slog.LevelInfo:
		return log_INFO
	case l < slog.LevelWarn:
		return log_NOTICE
	case l < slog.LevelError:
		return log_WARNING
	case l == slog.LevelError:
		return log_ERR
	case l == slog.LevelError+1:
		return log_CRIT
	case l == slog.LevelError+2:
		return log_ALERT
	default:
		return log_EMERG
	}
}
