package graylog

import (
	"compress/gzip"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net"
	"os"
	"strings"

	"github.com/mdigger/graylog/internal/buffer"
	"golang.org/x/exp/slog"
)

// Logger is an io.Logger for sending log messages to the Graylog server.
type Logger struct {
	*slog.Logger
	conn     net.Conn
	isUDP    bool
	host     string
	facility string
}

// Dial establishes a connection to the Graylog server and returns Logger to
// send log messages.
//
// Supported networks are "tcp", "tcp4" (IPv4-only), "tcp6" (IPv6-only), "udp",
// "udp4" (IPv4-only), "udp6" (IPv6-only).
// When using UDP as a transport layer, the messages sent are compressed using
// GZIP and automatically chunked.
//
// Attrs specify a list of attributes that will be added to all messages sent
// to Graylog server.
func Dial(network, address string, attrs ...slog.Attr) (*Logger, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}

	w := Logger{
		conn:     conn,
		isUDP:    strings.HasPrefix(network, "udp"),
		host:     host,
		facility: strings.TrimSpace(Facility), // copy to handler,
	}

	var handler slog.Handler = handler{w: w}
	if len(attrs) > 0 {
		handler = handler.WithAttrs(attrs)
	}
	w.Logger = slog.New(handler)

	return &w, nil
}

// Close closes a connection to the Graylog server.
func (w Logger) Close() error {
	return w.conn.Close()
}

// LogValue return log value with connection info (network & address).
func (w Logger) LogValue() slog.Value {
	if w.conn == nil {
		return slog.Value{}
	}

	addr := w.conn.RemoteAddr()
	return slog.GroupValue(
		slog.String("network", addr.Network()),
		slog.String("address", addr.String()),
	)
}

// ErrMessageToLarge returns when trying to send too long message other UDP.
var ErrMessageToLarge = errors.New("message too large")

var debugOutput = false // output gelf json before send

// write send a GELF message to the server.
func (w Logger) write(b []byte) error {
	if len(b) == 0 {
		return nil
	}

	// debug output
	if debugOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(json.RawMessage(b)); err != nil {
			return err
		}
	}

	// send other TCP
	if !w.isUDP {
		_, err := w.conn.Write(append(b, '\x00'))
		return err
	}

	// else send other UDP

	// Compress message
	buf := buffer.New()
	defer buf.Free()

	zw := gzip.NewWriter(buf)
	if _, err := zw.Write(b); err != nil {
		zw.Close()
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}

	// Some Graylog components are limited to processing up to 8192 bytes.
	const maxSize = 8192
	if len(*buf) <= maxSize {
		_, err := w.conn.Write(*buf)
		return err
	}

	// A message MUST NOT consist of more than 128 chunks.
	count := uint8((len(*buf)-1)/(maxSize-12) + 1)
	if count > 128 {
		return ErrMessageToLarge
	}

	// Prepend the following structure to your GELF message to make it chunked:
	// 	- Chunked GELF magic bytes - 2 bytes: 0x1e 0x0f
	// 	- Message ID - 8 bytes: Must be the same for every chunk of this
	// 	  message. Identifies the whole message and is used to reassemble the
	// 	  chunks later. Generate from millisecond timestamp + hostname, for
	// 	  example.
	// 	- Sequence number - 1 byte: The sequence number of this chunk starts
	// 	  at 0 and is always less than the sequence count.
	// 	- Sequence count - 1 byte: Total number of chunks this message has.
	header := [maxSize]byte{0x1e, 0x0f, 11: count}
	if _, err := rand.Read(header[2:10]); err != nil {
		return err
	}

	// All chunks MUST arrive within 5 seconds or the server will discard all
	// chunks that have arrived or are in the process of arriving.
	for i := uint8(0); i < count; i++ {
		header[10] = i
		n := copy(header[12:], *buf)
		if n == 0 {
			break
		}

		if _, err := w.conn.Write(header[:n+12]); err != nil {
			return err
		}
	}

	return nil
}
