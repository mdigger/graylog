package graylog

import (
	"fmt"
	"strings"

	"golang.org/x/exp/slog"
)

type handler struct {
	w     *Logger
	group string
	attrs []byte
}

// Enabled reports whether the handler handles records at the slog.DebugLevel.
func (h *handler) Enabled(l slog.Level) bool {
	return l >= slog.DebugLevel
}

// Handle send the log Record to Graylog server.
func (h *handler) Handle(r slog.Record) error {
	buf := newBuffer()
	defer bufPool.Put(buf)

	writeMessage(buf, level(r.Level), r.Message, r.Time, func() {
		if file, line := r.SourceLine(); line != 0 {
			writeAttrValue(buf, "file", fmt.Sprint(file, ":", line))
		}

		if len(h.attrs) > 0 {
			buf.Write(h.attrs)
		}

		r.Attrs(func(attr slog.Attr) {
			writeAttr(buf, attr, h.group)
		})
	})

	return h.w.write(buf.Bytes())
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	buf := newBuffer()
	defer bufPool.Put(buf)

	for _, attr := range attrs {
		writeAttr(buf, attr, h.group)
	}

	return &handler{
		w:     h.w,
		group: h.group,
		attrs: append(h.attrs, buf.Bytes()...),
	}
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
// The keys of all subsequent attributes, whether added by With or in a
// Record, should be qualified by the sequence of group names.
func (h *handler) WithGroup(name string) slog.Handler {
	if len(name) == 0 {
		return h
	}

	if h.group != "" {
		name = strings.Join([]string{h.group, name}, "_")
	}

	return &handler{
		w:     h.w,
		group: name,
		attrs: h.attrs,
	}
}

var _ slog.Handler = (*handler)(nil)
