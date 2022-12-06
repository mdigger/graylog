package graylog

import (
	"sync/atomic"

	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

type multiHandler struct{ ws atomic.Value }

// handlers returns a new slog.Handler that writes to all the specified
// handlers.
func multi(hs ...slog.Handler) slog.Handler {
	switch len(hs) {
	case 0:
		return slog.Default().Handler()
	case 1:
		return hs[0]
	default:
		lw := multiHandler{}
		lw.ws.Store(hs)
		return &lw
	}
}

func (mh multiHandler) Handle(r slog.Record) error {
	var firstErr error
	for _, h := range mh.handlers() {
		if err := h.Handle(r); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (mh multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return mh.withClone(func(h slog.Handler) slog.Handler {
		return h.WithAttrs(attrs)
	})
}

func (mh multiHandler) WithGroup(name string) slog.Handler {
	return mh.withClone(func(h slog.Handler) slog.Handler {
		return h.WithGroup(name)
	})
}

func (mh multiHandler) Enabled(level slog.Level) bool {
	for _, h := range mh.handlers() {
		if h.Enabled(level) {
			return true
		}
	}

	return false
}

func (mh multiHandler) handlers() []slog.Handler {
	return mh.ws.Load().([]slog.Handler)
}

func (mh multiHandler) withClone(f func(h slog.Handler) slog.Handler) slog.Handler {
	handlers := slices.Clone(mh.handlers())
	for i, h := range handlers {
		handlers[i] = f(h)
	}

	return multi(handlers...)
}
