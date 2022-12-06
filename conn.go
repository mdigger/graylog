package graylog

import (
	"compress/gzip"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"strings"

	"golang.org/x/exp/slog"
)

// Logger is an io.Logger for sending log messages to the Graylog server.
type Logger struct {
	*slog.Logger
	conn  net.Conn
	isUDP bool
}

// Dial establishes a connection to the Graylog server and returns Writer to
// send log messages.
//
// Supported networks are "tcp", "tcp4" (IPv4-only), "tcp6" (IPv6-only), "udp",
// "udp4" (IPv4-only), "udp6" (IPv6-only).
//
// The address has the form "host:port". The host must be a literal IP address,
// or a host name that can be resolved to IP addresses. The port must be a
// literal port number or a service name. If the host is a literal IPv6 address
// it must be enclosed in square brackets, as in "[2001:db8::1]:80" or
// "[fe80::1%zone]:80". The zone specifies the scope of the literal IPv6
// address as defined in RFC 4007. The functions net.JoinHostPort and
// net.SplitHostPort manipulate a pair of host and port in this form.
//
// When using TCP, and the host resolves to multiple IP addresses, Dial will
// try each IP address in order until one succeeds.
//
// Example:
//
//	Dial("udp", "198.51.100.1:12201")
//	Dial("udp", "[2001:db8::1]:12201")
//	Dial("tcp", "198.51.100.1:12201")
//
// When using UDP as a transport layer, the messages sent are compressed using
// GZIP and automatically chunked.
func Dial(network, address string) (*Logger, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}

	w := &Logger{
		conn:  conn,
		isUDP: strings.HasPrefix(network, "udp"),
	}

	w.Logger = slog.New(&handler{w: w})

	return w, nil
}

// Close closes a connection to the Graylog server.
func (w *Logger) Close() error {
	return w.conn.Close()
}

// LogValue return log value with connection info.
func (w *Logger) LogValue() slog.Value {
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

// write send a GELF message to the server.
func (w *Logger) write(b []byte) error {
	if len(b) == 0 {
		return nil
	}

	// FIXME: debug
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(json.RawMessage(b)); err != nil {
		return err
	}

	// send other TCP
	if !w.isUDP {
		_, err := w.conn.Write(append(b, '\x00'))
		return err
	}

	// else send other UDP

	// Compress message
	buf := newBuffer()
	defer bufPool.Put(buf)

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
	if buf.Len() <= maxSize {
		_, err := buf.WriteTo(w.conn)
		return err
	}

	// A message MUST NOT consist of more than 128 chunks.
	count := uint8((buf.Len()-1)/(maxSize-12) + 1)
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
		n, err := buf.Read(header[12:])
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}

		_, err = w.conn.Write(header[:n+12])
		if err != nil {
			return err
		}
	}

	return nil
}
