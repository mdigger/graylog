package graylog

import (
	"strconv"
	"time"

	"github.com/mdigger/graylog/internal/buffer"
	"golang.org/x/exp/slog"
)

type handler struct {
	w     Logger
	group string
	attrs []byte
}

// Enabled reports whether the handler handles records at the slog.DebugLevel.
func (h handler) Enabled(l slog.Level) bool {
	return l >= slog.LevelDebug
}

// Handle send the log Record to Graylog server.
func (h handler) Handle(r slog.Record) error {
	buf := buffer.New()
	defer buf.Free()

	// GELF header
	buf.WriteString(`{"version":"1.1","host":`)
	buf.WriteString(strconv.QuoteToASCII(h.w.host))

	// truncate short message
	title := truncate(r.Message, 120, 60)
	if title == "" {
		title = r.Message
	}
	buf.WriteString(`,"short_message":`)
	buf.WriteString(strconv.Quote(title))

	// write full message if different from title
	if title != r.Message {
		buf.WriteString(`,"full_message":`)
		buf.WriteString(strconv.Quote(r.Message))
	}

	// timestamp if defined
	if !r.Time.IsZero() {
		buf.WriteString(`,"timestamp":`)
		buf.WriteString(strconv.FormatFloat(
			float64(r.Time.UnixNano())/float64(time.Second), 'f', -1, 64))
	}

	// level
	buf.WriteString(`,"level":`)
	buf.WriteString(strconv.FormatUint(uint64(level(r.Level)), 10))

	// facility if defined
	if h.w.facility != "" {
		buf.WriteString(`,"_facility":`)
		buf.WriteString(strconv.Quote(h.w.facility))
	}

	// add source file info on warning and errors only
	if r.Level >= slog.LevelWarn {
		if file, line := r.SourceLine(); line != 0 {
			_ = writeAttrValue(buf, "file", file+":"+strconv.Itoa(line))
		}
	}

	// add stored attributes
	if len(h.attrs) > 0 {
		_, _ = buf.Write(h.attrs)
	}

	// add record attributes
	r.Attrs(func(attr slog.Attr) {
		writeAttr(buf, attr, h.group)
	})

	buf.WriteByte('}') // the end

	return h.w.write(*buf)
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
func (h handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	buf := buffer.New()
	defer buf.Free()

	for i := range attrs {
		writeAttr(buf, attrs[i], h.group)
	}

	return handler{
		w:     h.w,
		group: h.group,
		attrs: append(h.attrs, *buf...),
	}
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
// The keys of all subsequent attributes, whether added by With or in a
// Record, should be qualified by the sequence of group names.
func (h handler) WithGroup(name string) slog.Handler {
	if len(name) == 0 {
		return h
	}

	if h.group != "" {
		name = h.group + "_" + name
	}

	return handler{
		w:     h.w,
		group: name,
		attrs: h.attrs,
	}
}

var _ slog.Handler = (*handler)(nil)
