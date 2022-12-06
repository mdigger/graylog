package graylog

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/slog"
)

func writeAttr(w *bytes.Buffer, attr slog.Attr, prefix string) {
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

func writeAttrValue(w *bytes.Buffer, name string, value any) error {
	switch name {
	case "":
		return nil
	case "id":
		return errors.New(`unsupported attribute name "id"`)
	}

	w.WriteString(`,"`)
	w.WriteString(fixName(name))
	w.WriteString(`":`)

	var (
		wquote = func(v string) { w.WriteString(strconv.Quote(v)) }
		wint   = func(v int64) { w.WriteString(strconv.FormatInt(v, 10)) }
		wuint  = func(v uint64) { w.WriteString(strconv.FormatUint(v, 10)) }
		wjson  = func(v any) error {
			b, err := json.Marshal(v)
			if err != nil {
				return err
			}
			w.Write(b)
			return nil
		}
		wtext = func(v encoding.TextMarshaler) error {
			b, err := v.MarshalText()
			if err != nil {
				return err
			}
			wquote(string(b))
			return nil
		}
		wfloat = func(v float64) error {
			switch {
			case math.IsInf(v, 1):
				w.WriteString(`"+Inf"`)
			case math.IsInf(v, -1):
				w.WriteString(`"-Inf"`)
			case math.IsNaN(v):
				w.WriteString(`"NaN"`)
			default:
				return wjson(v)
			}
			return nil
		}
	)

	switch v := value.(type) {
	case nil:
		w.WriteString("null")
	case string:
		wquote(v)
	case int:
		wint(int64(v))
	case int8:
		wint(int64(v))
	case int16:
		wint(int64(v))
	case int32:
		wint(int64(v))
	case int64:
		wint(v)
	case uint:
		wuint(uint64(v))
	case uint8:
		wuint(uint64(v))
	case uint16:
		wuint(uint64(v))
	case uint32:
		wuint(uint64(v))
	case uint64:
		wuint(v)
	case float32:
		return wfloat(float64(v))
	case float64:
		return wfloat(v)
	case bool:
		w.WriteByte('"')
		w.WriteString(strconv.FormatBool(v))
		w.WriteByte('"')
	case time.Duration:
		w.WriteByte('"')
		w.WriteString(v.String())
		w.WriteByte('"')
	case time.Time:
		if v.IsZero() {
			w.WriteString(`""`)
			break
		}

		w.WriteByte('"')
		w.WriteString(v.String())
		w.WriteByte('"')
	case error:
		wquote(v.Error())
	case fmt.Stringer:
		wquote(v.String())
	case encoding.TextMarshaler:
		return wtext(v)
	default:
		return wjson(v)
	}

	return nil
}

var reName = regexp.MustCompile(`([^\w\.\-]+)`)

func fixName(s string) string {
	s = strings.TrimSpace(s)
	s = reName.ReplaceAllStringFunc(s, func(s string) string {
		var buf strings.Builder
		for _, r := range []rune(s) {
			switch {
			case r <= '\x7F':
				buf.WriteRune('_')
			default:
				fmt.Fprintf(&buf, "\\u%04x", r)
			}
		}
		return buf.String()
	})
	return strings.Join([]string{"_", s}, "")
}
