package graylog

import (
	"encoding"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mdigger/graylog/internal/buffer"
	"golang.org/x/exp/slog"
)

func writeAttr(w *buffer.Buffer, attr slog.Attr, prefix string) {
	if attr.Key == "" {
		return
	}

	name := attr.Key
	if prefix != "" {
		name = strings.Join([]string{prefix, name}, "_")
	}

	switch attr.Value.Kind() {
	case slog.GroupKind:
		for _, attr := range attr.Value.Group() {
			writeAttr(w, attr, name)
		}
	case slog.LogValuerKind:
		attr := slog.Attr{Key: name, Value: attr.Value.Resolve()}
		writeAttr(w, attr, "")
	default:
		_ = writeAttrValue(w, name, attr.Value.Any())
	}
}

func writeAttrValue(w *buffer.Buffer, name string, value any) error {
	switch name {
	case "":
		return nil
	case "id":
		return errors.New(`unsupported attribute name "id"`)
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
	buf := buffer.New()
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

	return strings.Join([]string{"_", s}, "")
}
