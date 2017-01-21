package logstash

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
	"unicode/utf8"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

type encoder struct {
	buf     *bytes.Buffer
	scratch [64]byte
}

var hex = "0123456789abcdef"

func makeLogstashEventEncoder(beat string) (func(interface{}) ([]byte, error), error) {
	enc := encoder{buf: bytes.NewBuffer(nil)}

	beatName, err := json.Marshal(beat)
	if err != nil {
		return nil, err
	}

	cb := func(rawData interface{}) ([]byte, error) {
		event := rawData.(outputs.Data).Event
		buf := enc.buf
		buf.Reset()

		buf.WriteRune('{')
		if _, hasMeta := event["@metadata"]; !hasMeta {
			typ := event["type"].(string)
			buf.WriteString(`"@metadata":{"type":`)
			encodeString(buf, typ)
		}

		buf.WriteString(`,"beat":`)
		buf.Write(beatName)
		buf.WriteString(`},`)
		err := enc.encodeKeyValues(event)
		if err != nil {
			logp.Err("jsonEncode failed with: %v", err)
			return nil, err
		}

		b := buf.Bytes()
		b[len(b)-1] = '}'

		return buf.Bytes(), nil
	}

	return cb, nil
}

func (enc *encoder) encodeKeyValues(event common.MapStr) error {
	buf := enc.buf

	for k, v := range event {
		encodeString(buf, k)
		buf.WriteRune(':')

		switch val := v.(type) {
		case common.MapStr:
			buf.WriteRune('{')
			if len(val) == 0 {
				buf.WriteRune('}')
			} else {
				if err := enc.encodeKeyValues(val); err != nil {
					return err
				}

				b := buf.Bytes()
				b[len(b)-1] = '}'
			}
		case bool:
			if val {
				buf.WriteString("true")
			} else {
				buf.WriteString("false")
			}

		case int8:
			enc.encodeInt(int64(val))
		case int16:
			enc.encodeInt(int64(val))
		case int32:
			enc.encodeInt(int64(val))
		case int64:
			enc.encodeInt(val)

		case uint8:
			enc.encodeUint(uint64(val))
		case uint16:
			enc.encodeUint(uint64(val))
		case uint32:
			enc.encodeUint(uint64(val))
		case uint64:
			enc.encodeUint(val)

		case float32:
			enc.encodeFloat(float64(val))
		case float64:
			enc.encodeFloat(val)

		default:
			// fallback to json.Marshal
			tmp, err := json.Marshal(v)
			if err != nil {
				return err
			}
			buf.Write(tmp)
		}

		buf.WriteRune(',')
	}

	return nil
}

func (enc *encoder) encodeInt(i int64) {
	b := strconv.AppendInt(enc.scratch[:0], i, 10)
	enc.buf.Write(b)
}

func (enc *encoder) encodeUint(u uint64) {
	b := strconv.AppendUint(enc.scratch[:0], u, 10)
	enc.buf.Write(b)
}

func (enc *encoder) encodeFloat(f float64) {
	switch {
	case math.IsInf(f, 0):
		enc.buf.WriteString("Inf")
	case math.IsNaN(f):
		enc.buf.WriteString("NaN")
	default:
		b := strconv.AppendFloat(enc.scratch[:0], f, 'g', -1, 64)
		enc.buf.Write(b)
	}
}

// JSON string encoded copied from "json" package
func encodeString(buf *bytes.Buffer, s string) {
	buf.WriteByte('"')
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if 0x20 <= b && b != '\\' && b != '"' && b != '<' && b != '>' && b != '&' {
				i++
				continue
			}
			if start < i {
				buf.WriteString(s[start:i])
			}
			switch b {
			case '\\', '"':
				buf.WriteByte('\\')
				buf.WriteByte(b)
			case '\n':
				buf.WriteByte('\\')
				buf.WriteByte('n')
			case '\r':
				buf.WriteByte('\\')
				buf.WriteByte('r')
			case '\t':
				buf.WriteByte('\\')
				buf.WriteByte('t')
			default:
				// This encodes bytes < 0x20 except for \n and \r,
				// as well as <, > and &. The latter are escaped because they
				// can lead to security holes when user-controlled strings
				// are rendered into JSON and served to some browsers.
				buf.WriteString(`\u00`)
				buf.WriteByte(hex[b>>4])
				buf.WriteByte(hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				buf.WriteString(s[start:i])
			}
			buf.WriteString(`\ufffd`)
			i += size
			start = i
			continue
		}
		// U+2028 is LINE SEPARATOR.
		// U+2029 is PARAGRAPH SEPARATOR.
		// They are both technically valid characters in JSON strings,
		// but don't work in JSONP, which has to be evaluated as JavaScript,
		// and can lead to security holes there. It is valid JSON to
		// escape them, so we do so unconditionally.
		// See http://timelessrepo.com/json-isnt-a-javascript-subset for discussion.
		if c == '\u2028' || c == '\u2029' {
			if start < i {
				buf.WriteString(s[start:i])
			}
			buf.WriteString(`\u202`)
			buf.WriteByte(hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		buf.WriteString(s[start:])
	}
	buf.WriteByte('"')
}
