package graylog

import (
	"encoding"
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/exp/slog"
)

func writeAttr(w *buffer, attr slog.Attr, prefix string) {
	if attr.Key == "" {
		return
	}

	name := attr.Key
	if prefix != "" {
		name = prefix + "_" + name
	}

	switch attr.Value.Kind() {
	case slog.GroupKind:
		attrs := attr.Value.Group()
		for i := range attrs {
			writeAttr(w, attrs[i], name)
		}
	case slog.LogValuerKind:
		attr := slog.Attr{Key: name, Value: attr.Value.Resolve()}
		writeAttr(w, attr, "")
	default:
		_ = writeAttrValue(w, name, attr.Value.Any())
	}
}

func writeAttrValue(w *buffer, name string, value any) error {
	switch name {
	case "":
		return nil
	case "id", "source", "message", "full_message", "level", "timestamp", "facility", "file":
		name += "_" // добавляем уникальность уже используемым названиям полей graylog
	}

	w.WriteString(`,"`)
	w.WriteString(fixName(name))
	w.WriteString(`":`)

	switch v := value.(type) {
	case nil:
		w.WriteString("null")
	case string:
		w.WriteQuoted(v)
	case int:
		w.WriteInt(int64(v))
	case int8:
		w.WriteInt(int64(v))
	case int16:
		w.WriteInt(int64(v))
	case int32:
		w.WriteInt(int64(v))
	case int64:
		w.WriteInt(v)
	case uint:
		w.WriteUint(uint64(v))
	case uint8:
		w.WriteUint(uint64(v))
	case uint16:
		w.WriteUint(uint64(v))
	case uint32:
		w.WriteUint(uint64(v))
	case uint64:
		w.WriteUint(v)
	case float32:
		return w.WriteFloat(float64(v))
	case float64:
		return w.WriteFloat(v)
	case bool:
		w.WriteBool(v)
	case time.Duration:
		w.WriteQuoted(v.String())
	case time.Time:
		if v.IsZero() {
			w.WriteString(`""`)
			break
		}

		w.WriteQuoted(v.String())
	case error:
		w.WriteQuoted(v.Error())
	case fmt.Stringer:
		w.WriteQuoted(v.String())
	case encoding.TextMarshaler:
		return w.WriteText(v)
	default:
		return w.WriteJson(v)
	}

	return nil
}

var reName = regexp.MustCompile(`([^\w\.\-]+)`)

func fixName(s string) string {
	buf := newBuffer()
	defer buf.Free()

	s = strings.TrimSpace(s)
	s = reName.ReplaceAllStringFunc(s, func(s string) string {
		buf.Reset()

		for _, r := range s {
			switch {
			case r <= '\x7F':
				buf.WriteByte('_')
			default:
				buf.WriteString(fmt.Sprintf("\\u%04x", r)) // FIXME: r escapes to heap
			}
		}

		return buf.String()
	})

	return "_" + s
}
