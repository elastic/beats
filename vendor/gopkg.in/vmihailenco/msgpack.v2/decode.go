package msgpack

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"time"

	"gopkg.in/vmihailenco/msgpack.v2/codes"
)

const bytesAllocLimit = 1024 * 1024 // 1mb

type bufReader interface {
	Read([]byte) (int, error)
	ReadByte() (byte, error)
	UnreadByte() error
}

func newBufReader(r io.Reader) bufReader {
	if br, ok := r.(bufReader); ok {
		return br
	}
	return bufio.NewReader(r)
}

func makeBuffer() []byte {
	return make([]byte, 0, 64)
}

// Unmarshal decodes the MessagePack-encoded data and stores the result
// in the value pointed to by v.
func Unmarshal(data []byte, v ...interface{}) error {
	return NewDecoder(bytes.NewReader(data)).Decode(v...)
}

type Decoder struct {
	DecodeMapFunc func(*Decoder) (interface{}, error)

	r   bufReader
	buf []byte

	extLen int
	rec    []byte // accumulates read data if not nil
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		DecodeMapFunc: decodeMap,

		r:   newBufReader(r),
		buf: makeBuffer(),
	}
}

func (d *Decoder) Reset(r io.Reader) error {
	d.r = newBufReader(r)
	return nil
}

func (d *Decoder) Decode(v ...interface{}) error {
	for _, vv := range v {
		if err := d.decode(vv); err != nil {
			return err
		}
	}
	return nil
}

func (d *Decoder) decode(dst interface{}) error {
	var err error
	switch v := dst.(type) {
	case *string:
		if v != nil {
			*v, err = d.DecodeString()
			return err
		}
	case *[]byte:
		if v != nil {
			return d.decodeBytesPtr(v)
		}
	case *int:
		if v != nil {
			*v, err = d.DecodeInt()
			return err
		}
	case *int8:
		if v != nil {
			*v, err = d.DecodeInt8()
			return err
		}
	case *int16:
		if v != nil {
			*v, err = d.DecodeInt16()
			return err
		}
	case *int32:
		if v != nil {
			*v, err = d.DecodeInt32()
			return err
		}
	case *int64:
		if v != nil {
			*v, err = d.DecodeInt64()
			return err
		}
	case *uint:
		if v != nil {
			*v, err = d.DecodeUint()
			return err
		}
	case *uint8:
		if v != nil {
			*v, err = d.DecodeUint8()
			return err
		}
	case *uint16:
		if v != nil {
			*v, err = d.DecodeUint16()
			return err
		}
	case *uint32:
		if v != nil {
			*v, err = d.DecodeUint32()
			return err
		}
	case *uint64:
		if v != nil {
			*v, err = d.DecodeUint64()
			return err
		}
	case *bool:
		if v != nil {
			*v, err = d.DecodeBool()
			return err
		}
	case *float32:
		if v != nil {
			*v, err = d.DecodeFloat32()
			return err
		}
	case *float64:
		if v != nil {
			*v, err = d.DecodeFloat64()
			return err
		}
	case *[]string:
		return d.decodeStringSlicePtr(v)
	case *map[string]string:
		return d.decodeMapStringStringPtr(v)
	case *map[string]interface{}:
		return d.decodeMapStringInterfacePtr(v)
	case *time.Duration:
		if v != nil {
			vv, err := d.DecodeInt64()
			*v = time.Duration(vv)
			return err
		}
	case *time.Time:
		if v != nil {
			*v, err = d.DecodeTime()
			return err
		}
	}

	v := reflect.ValueOf(dst)
	if !v.IsValid() {
		return errors.New("msgpack: Decode(nil)")
	}
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("msgpack: Decode(nonsettable %T)", dst)
	}
	v = v.Elem()
	if !v.IsValid() {
		return fmt.Errorf("msgpack: Decode(nonsettable %T)", dst)
	}
	return d.DecodeValue(v)
}

func (d *Decoder) DecodeValue(v reflect.Value) error {
	decode := getDecoder(v.Type())
	return decode(d, v)
}

func (d *Decoder) DecodeNil() error {
	c, err := d.readByte()
	if err != nil {
		return err
	}
	if c != codes.Nil {
		return fmt.Errorf("msgpack: invalid code %x decoding nil", c)
	}
	return nil
}

func (d *Decoder) DecodeBool() (bool, error) {
	c, err := d.readByte()
	if err != nil {
		return false, err
	}
	return d.bool(c)
}

func (d *Decoder) bool(c byte) (bool, error) {
	if c == codes.False {
		return false, nil
	}
	if c == codes.True {
		return true, nil
	}
	return false, fmt.Errorf("msgpack: invalid code %x decoding bool", c)
}

func (d *Decoder) interfaceValue(v reflect.Value) error {
	vv, err := d.DecodeInterface()
	if err != nil {
		return err
	}
	if vv != nil {
		if v.Type() == errorType {
			if vv, ok := vv.(string); ok {
				v.Set(reflect.ValueOf(errors.New(vv)))
				return nil
			}
		}

		v.Set(reflect.ValueOf(vv))
	}
	return nil
}

// DecodeInterface decodes value into interface. Possible value types are:
//   - nil,
//   - bool,
//   - int64 for negative numbers,
//   - uint64 for positive numbers,
//   - float32 and float64,
//   - string,
//   - slices of any of the above,
//   - maps of any of the above.
func (d *Decoder) DecodeInterface() (interface{}, error) {
	c, err := d.readByte()
	if err != nil {
		return nil, err
	}

	if codes.IsFixedNum(c) {
		if int8(c) < 0 {
			return d.int(c)
		}
		return d.uint(c)
	}
	if codes.IsFixedMap(c) {
		d.r.UnreadByte()
		return d.DecodeMap()
	}
	if codes.IsFixedArray(c) {
		return d.decodeSlice(c)
	}
	if codes.IsFixedString(c) {
		return d.string(c)
	}

	switch c {
	case codes.Nil:
		return nil, nil
	case codes.False, codes.True:
		return d.bool(c)
	case codes.Float:
		return d.float32(c)
	case codes.Double:
		return d.float64(c)
	case codes.Uint8, codes.Uint16, codes.Uint32, codes.Uint64:
		return d.uint(c)
	case codes.Int8, codes.Int16, codes.Int32, codes.Int64:
		return d.int(c)
	case codes.Bin8, codes.Bin16, codes.Bin32:
		return d.bytes(c, nil)
	case codes.Str8, codes.Str16, codes.Str32:
		return d.string(c)
	case codes.Array16, codes.Array32:
		return d.decodeSlice(c)
	case codes.Map16, codes.Map32:
		d.r.UnreadByte()
		return d.DecodeMap()
	case codes.FixExt1, codes.FixExt2, codes.FixExt4, codes.FixExt8, codes.FixExt16,
		codes.Ext8, codes.Ext16, codes.Ext32:
		return d.ext(c)
	}

	return 0, fmt.Errorf("msgpack: unknown code %x decoding interface{}", c)
}

// Skip skips next value.
func (d *Decoder) Skip() error {
	c, err := d.readByte()
	if err != nil {
		return err
	}

	if codes.IsFixedNum(c) {
		return nil
	} else if codes.IsFixedMap(c) {
		return d.skipMap(c)
	} else if codes.IsFixedArray(c) {
		return d.skipSlice(c)
	} else if codes.IsFixedString(c) {
		return d.skipBytes(c)
	}

	switch c {
	case codes.Nil, codes.False, codes.True:
		return nil
	case codes.Uint8, codes.Int8:
		return d.skipN(1)
	case codes.Uint16, codes.Int16:
		return d.skipN(2)
	case codes.Uint32, codes.Int32, codes.Float:
		return d.skipN(4)
	case codes.Uint64, codes.Int64, codes.Double:
		return d.skipN(8)
	case codes.Bin8, codes.Bin16, codes.Bin32:
		return d.skipBytes(c)
	case codes.Str8, codes.Str16, codes.Str32:
		return d.skipBytes(c)
	case codes.Array16, codes.Array32:
		return d.skipSlice(c)
	case codes.Map16, codes.Map32:
		return d.skipMap(c)
	case codes.FixExt1, codes.FixExt2, codes.FixExt4, codes.FixExt8, codes.FixExt16, codes.Ext8, codes.Ext16, codes.Ext32:
		return d.skipExt(c)
	}

	return fmt.Errorf("msgpack: unknown code %x", c)
}

// peekCode returns next MessagePack code. See
// https://github.com/msgpack/msgpack/blob/master/spec.md#formats for details.
func (d *Decoder) PeekCode() (code byte, err error) {
	code, err = d.r.ReadByte()
	if err != nil {
		return 0, err
	}
	return code, d.r.UnreadByte()
}

func (d *Decoder) hasNilCode() bool {
	code, err := d.PeekCode()
	return err == nil && code == codes.Nil
}

func (d *Decoder) readByte() (byte, error) {
	c, err := d.r.ReadByte()
	if err != nil {
		return 0, err
	}
	if d.rec != nil {
		d.rec = append(d.rec, c)
	}
	return c, nil
}

func (d *Decoder) readFull(b []byte) error {
	_, err := io.ReadFull(d.r, b)
	if err != nil {
		return err
	}
	if d.rec != nil {
		d.rec = append(d.rec, b...)
	}
	return nil
}

func (d *Decoder) readN(n int) ([]byte, error) {
	buf, err := readN(d.r, d.buf, n)
	if err != nil {
		return nil, err
	}
	d.buf = buf
	if d.rec != nil {
		d.rec = append(d.rec, buf...)
	}
	return buf, nil
}

func readN(r io.Reader, b []byte, n int) ([]byte, error) {
	if n == 0 && b == nil {
		return make([]byte, 0), nil
	}

	if cap(b) >= n {
		b = b[:n]
		_, err := io.ReadFull(r, b)
		return b, err
	}
	b = b[:cap(b)]

	pos := 0
	for len(b) < n {
		diff := n - len(b)
		if diff > bytesAllocLimit {
			diff = bytesAllocLimit
		}
		b = append(b, make([]byte, diff)...)

		_, err := io.ReadFull(r, b[pos:])
		if err != nil {
			return nil, err
		}

		pos = len(b)
	}

	return b, nil
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
