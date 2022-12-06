package graylog

import (
	"golang.org/x/exp/slog"
)

// priority is a Graylog level.
type priority uint8

const (
	// From /usr/include/sys/syslog.h.
	log_EMERG priority = iota
	log_ALERT
	log_CRIT
	log_ERR
	log_WARNING
	log_NOTICE
	log_INFO
	log_DEBUG
)

func level(l slog.Level) priority {
	switch {
	case l < slog.DebugLevel:
		return log_DEBUG + priority(slog.DebugLevel-l.Level())
	case l < slog.InfoLevel:
		return log_DEBUG
	case l == slog.InfoLevel:
		return log_INFO
	case l < slog.WarnLevel:
		return log_NOTICE
	case l < slog.ErrorLevel:
		return log_WARNING
	case l == slog.ErrorLevel:
		return log_ERR
	case l == slog.ErrorLevel+1:
		return log_CRIT
	case l == slog.ErrorLevel+2:
		return log_ALERT
	default:
		return log_EMERG
	}
}
