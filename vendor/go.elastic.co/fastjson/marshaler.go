// Copyright 2018 Elasticsearch BV
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fastjson

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

// Marshaler defines an interface that types can implement to provide
// fast JSON marshaling.
type Marshaler interface {
	// MarshalFastJSON writes a JSON representation of the type to w.
	//
	// MarshalFastJSON is expected to suppress any panics. Depending
	// on the application, it may be expected that MarshalFastJSON
	// writes valid JSON to w, even in error cases.
	//
	// The returned error will be propagated up through to callers of
	// fastjson.Marshal.
	MarshalFastJSON(w *Writer) error
}

// Appender defines an interface that types can implement to append
// their JSON representation to a buffer.
type Appender interface {
	// AppendJSON appends the JSON representation of the value to the
	// buffer, and returns the extended buffer.
	//
	// AppendJSON is required not to panic or fail.
	AppendJSON([]byte) []byte
}

// Marshal marshals v as JSON to w.
//
// For all basic types, Marshal uses w's methods to marshal the values
// directly. If v implements Marshaler, its MarshalFastJSON method will
// be used; if v implements Appender, its AppendJSON method will be used,
// and it is assumed to append valid JSON. As a final resort, we use
// json.Marshal.
//
// Where json.Marshal is used internally (see above), errors or panics
// produced by json.Marshal will be encoded as JSON objects, with special keys
// "__ERROR__" for errors, and "__PANIC__" for panics. e.g. if json.Marshal
// panics due to a broken json.Marshaler implementation or assumption, then
// Marshal will encode the panic as
//
//     {"__PANIC__": "panic calling MarshalJSON for type Foo: reason"}
//
// Marshal returns the first error encountered.
func Marshal(w *Writer, v interface{}) error {
	switch v := v.(type) {
	case nil:
		w.RawString("null")
	case string:
		w.String(v)
	case uint:
		w.Uint64(uint64(v))
	case uint8:
		w.Uint64(uint64(v))
	case uint16:
		w.Uint64(uint64(v))
	case uint32:
		w.Uint64(uint64(v))
	case uint64:
		w.Uint64(v)
	case int:
		w.Int64(int64(v))
	case int8:
		w.Int64(int64(v))
	case int16:
		w.Int64(int64(v))
	case int32:
		w.Int64(int64(v))
	case int64:
		w.Int64(v)
	case float32:
		w.Float32(v)
	case float64:
		w.Float64(v)
	case bool:
		w.Bool(v)
	case map[string]interface{}:
		if v == nil {
			w.RawString("null")
			return nil
		}
		w.RawByte('{')
		var firstErr error
		first := true
		for k, v := range v {
			if first {
				first = false
			} else {
				w.RawByte(',')
			}
			w.String(k)
			w.RawByte(':')
			if err := Marshal(w, v); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		w.RawByte('}')
		return firstErr
	case Marshaler:
		return v.MarshalFastJSON(w)
	case Appender:
		w.buf = v.AppendJSON(w.buf)
	default:
		return marshalReflect(w, v)
	}
	return nil
}

func marshalReflect(w *Writer, v interface{}) (result error) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("%s", r)
			}
			result = errors.Wrapf(err, "panic calling MarshalJSON for type %T", v)
			w.RawString(`{"__PANIC__":`)
			w.String(fmt.Sprint(result))
			w.RawByte('}')
		}
	}()
	raw, err := json.Marshal(v)
	if err != nil {
		w.RawString(`{"__ERROR__":`)
		w.String(fmt.Sprint(err))
		w.RawByte('}')
		return err
	}
	w.RawBytes(raw)
	return nil
}
