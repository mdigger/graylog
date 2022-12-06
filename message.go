package graylog

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Used when generating a log message in GELF format.
var (
	Facility    = filepath.Base(os.Args[0])
	Hostname, _ = os.Hostname()
)

func writeMessage(w *bytes.Buffer, l priority, msg string, ts time.Time, extra func()) {
	if l > log_DEBUG {
		return
	}

	// GELF header
	w.WriteString(`{"version":"1.1","host":`)
	host := Hostname
	if host == "" {
		host = "localhost"
	}
	w.WriteString(strconv.QuoteToASCII(host))

	// truncate short message
	title := truncate(msg, 120, 60)
	if title == "" {
		title = msg
	}
	w.WriteString(`,"short_message":`)
	w.WriteString(strconv.Quote(title))

	// write full message if different from title
	if title != msg {
		w.WriteString(`,"full_message":`)
		w.WriteString(strconv.Quote(msg))
	}

	// timestamp if defined
	if !ts.IsZero() {
		w.WriteString(`,"timestamp":`)
		w.WriteString(strconv.FormatFloat(
			float64(ts.UnixNano())/float64(time.Second), 'f', -1, 64))
	}

	// facility if defined
	if Facility != "" {
		w.WriteString(`,"_facility":`)
		w.WriteString(strconv.Quote(Facility))
	}

	// add extra data
	if extra != nil {
		extra()
	}

	w.WriteByte('}') // the end
}
