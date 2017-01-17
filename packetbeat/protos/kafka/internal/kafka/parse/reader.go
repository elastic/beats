package parse

import (
	"github.com/elastic/beats/libbeat/common/streambuf"

	"github.com/elastic/beats/packetbeat/protos/kafka/internal/kafka"
)

type protoReader struct {
	streambuf.Buffer
}

func makeReader(payload []byte) *protoReader {
	return &protoReader{*streambuf.NewFixed(payload)}
}

// kafka protocol primitive types

func (r *protoReader) int8() (int8, bool) {
	v, err := r.ReadNetUint8()
	return int8(v), err == nil
}

func (r *protoReader) int16() (int16, bool) {
	v, err := r.ReadNetUint16()
	return int16(v), err == nil
}

func (r *protoReader) int32() (int32, bool) {
	v, err := r.ReadNetUint32()
	return int32(v), err == nil
}

func (r *protoReader) int64() (int64, bool) {
	v, err := r.ReadNetUint64()
	return int64(v), err == nil
}

func (r *protoReader) string() (string, bool) {
	tmp, ok := r.stringBytes()
	return string(tmp), ok
}

func (r *protoReader) stringBytes() ([]byte, bool) {
	if i, ok := r.int16(); ok {
		return r.collectBytes(int(i))
	}
	return nil, false
}

func (r *protoReader) bytes() ([]byte, bool) {
	if i, ok := r.int32(); ok {
		return r.collectBytes(int(i))
	}
	return nil, false
}

func (r *protoReader) collectBytes(len int) ([]byte, bool) {
	switch {
	case len == -1:
		return nil, true
	case len < 0:
		return nil, false
	}

	str, err := r.Collect(len)
	if err != nil {
		return nil, false
	}

	return str, true
}

// kafka protocol named types

func (r *protoReader) version() (kafka.APIVersion, bool) {
	v, ok := r.int16()
	return kafka.APIVersion(v), ok
}

func (r *protoReader) id() (kafka.ID, bool) {
	v, ok := r.int32()
	return kafka.ID(v), ok
}

func (r *protoReader) err() (kafka.ErrorCode, bool) {
	v, ok := r.int16()
	return kafka.ErrorCode(v), ok
}

func (r *protoReader) messageSet() (kafka.RawMessageSet, bool) {
	payload, ok := r.bytes()
	return kafka.RawMessageSet{payload}, ok
}

// kafka protocol array/map types

func (r *protoReader) int32Arr() (arr []int32, ok bool) {
	ok = r.arr(func() {
		if v, ok := r.int32(); ok {
			arr = append(arr, v)
		}
	})
	return
}

func (r *protoReader) int64Arr() (arr []int64, ok bool) {
	ok = r.arr(func() {
		if v, ok := r.int64(); ok {
			arr = append(arr, v)
		}
	})
	return
}

func (r *protoReader) idArr() (arr []kafka.ID, ok bool) {
	ok = r.arr(func() {
		if id, ok := r.id(); ok {
			arr = append(arr, id)
		}
	})
	return
}

func (r *protoReader) arr(f func()) bool {
	n, ok := r.int32()
	if !ok {
		return false
	}

	for ; n > 0; n-- {
		f()
		if r.Failed() {
			return false
		}
	}

	return true
}

func (r *protoReader) stringMap(f func(string)) bool {
	n, ok := r.int32()
	if !ok {
		return false
	}

	for ; n > 0; n-- {
		key, ok := r.string()
		if !ok {
			return false
		}

		f(key)
		if r.Failed() {
			return false
		}
	}

	return true
}

func (r *protoReader) stringMetaMap() (map[string][]byte, bool) {
	m := map[string][]byte{}
	ok := r.stringMap(func(name string) {
		if meta, ok := r.bytes(); ok {
			m[name] = meta
		}
	})
	if !ok {
		return nil, false
	}

	return m, true
}

func (r *protoReader) idMap(f func(kafka.ID)) bool {
	n, ok := r.int32()
	if !ok {
		return false
	}

	for ; n > 0; n-- {
		id, ok := r.id()
		if !ok {
			return false
		}

		f(id)
		if r.Failed() {
			return false
		}
	}

	return true
}
