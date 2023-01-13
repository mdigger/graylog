// Package buffer provides a pool-allocated byte buffer with custom methods.
package graylog

import (
	"encoding"
	"encoding/json"
	"math"
	"strconv"
	"sync"
)

// buffer adapted from go/src/fmt/print.go
type buffer []byte

// Having an initial size gives a dramatic speedup.
var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 1024)
		return (*buffer)(&b)
	},
}

func newBuffer() *buffer {
	return bufPool.Get().(*buffer)
}

func (b *buffer) Free() {
	// To reduce peak allocation, return only smaller buffers to the pool.
	const maxBufferSize = 16 << 10
	if cap(*b) <= maxBufferSize {
		*b = (*b)[:0]
		bufPool.Put(b)
	}
}

func (b *buffer) Reset() {
	*b = (*b)[:0]
}

func (b *buffer) Write(p []byte) (int, error) {
	*b = append(*b, p...)
	return len(p), nil
}

func (b *buffer) WriteString(s string) {
	*b = append(*b, s...)
}

func (b *buffer) WriteQuoted(s string) {
	*b = strconv.AppendQuote(*b, s)
}

func (b *buffer) WriteByte(c byte) {
	*b = append(*b, c)
}

func (b *buffer) WriteBool(v bool) {
	c := `"false"`
	if v {
		c = `"true"`
	}
	*b = append(*b, c...)
}

func (b *buffer) WriteInt(i int64) {
	*b = strconv.AppendInt(*b, i, 10)
}

func (b *buffer) WriteUint(i uint64) {
	*b = strconv.AppendUint(*b, i, 10)
}

func (b *buffer) WriteJson(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, _ = b.Write(data)
	return nil
}

func (b *buffer) WriteText(v encoding.TextMarshaler) error {
	data, err := v.MarshalText()
	if err != nil {
		return err
	}
	b.WriteQuoted(string(data))
	return nil
}

func (b *buffer) WriteFloat(v float64) error {
	switch {
	case math.IsInf(v, 1):
		b.WriteString(`"+Inf"`)
	case math.IsInf(v, -1):
		b.WriteString(`"-Inf"`)
	case math.IsNaN(v):
		b.WriteString(`"NaN"`)
	default:
		return b.WriteJson(v)
	}
	return nil
}

// func (b *Buffer) WritePosInt(i int) {
// 	b.WritePosIntWidth(i, 0)
// }

// // WritePosIntWidth writes non-negative integer i to the buffer, padded on the left
// // by zeroes to the given width. Use a width of 0 to omit padding.
// func (b *Buffer) WritePosIntWidth(i, width int) {
// 	// Cheap integer to fixed-width decimal ASCII.
// 	// Copied from log/log.go.

// 	if i < 0 {
// 		panic("negative int")
// 	}

// 	// Assemble decimal in reverse order.
// 	var bb [20]byte
// 	bp := len(bb) - 1
// 	for i >= 10 || width > 1 {
// 		width--
// 		q := i / 10
// 		bb[bp] = byte('0' + i - q*10)
// 		bp--
// 		i = q
// 	}
// 	// i < 10
// 	bb[bp] = byte('0' + i)
// 	b.Write(bb[bp:])
// }

func (b *buffer) String() string {
	return string(*b)
}
