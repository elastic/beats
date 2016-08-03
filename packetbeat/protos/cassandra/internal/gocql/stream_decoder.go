package cassandra

import (
	"errors"
	"fmt"
	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/elastic/beats/libbeat/logp"
	"net"
)

type StreamDecoder struct {
	r *streambuf.Buffer
}

func (f StreamDecoder) ReadHeader(r *streambuf.Buffer) (head *frameHeader, err error) {
	v, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	version := v & protoVersionMask

	if version < protoVersion1 || version > protoVersion4 {
		return nil, fmt.Errorf("unsupported response version: %d", version)
	}

	head = &frameHeader{}

	head.Version = protoVersion(v)

	head.Flags, err = r.ReadByte()

	if version > protoVersion2 {
		stream, err := r.ReadNetUint16()
		if err != nil {
			return nil, err
		}
		head.Stream = int(stream)

		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}

		head.Op = FrameOp(b)
		l, err := r.ReadNetUint32()
		if err != nil {
			return nil, err
		}
		head.Length = int(l)
	} else {
		stream, err := r.ReadNetUint8()
		if err != nil {
			return nil, err
		}
		head.Stream = int(stream)

		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}

		head.Op = FrameOp(b)
		l, err := r.ReadNetUint32()
		if err != nil {
			return nil, err
		}
		head.Length = int(l)
	}

	if head.Length < 0 {
		return nil, fmt.Errorf("frame body length can not be less than 0: %d", head.Length)
	} else if head.Length > maxFrameSize {
		// need to free up the connection to be used again
		logp.Err("head length is too large")
		return nil, ErrFrameTooBig
	}

	if !r.Avail(head.Length) {
		return nil, errors.New(fmt.Sprintf("frame length is not enough as expected length: %v", head.Length))
	}

	logp.Debug("cassandra", "header: %v", head)

	return head, nil
}

func (f StreamDecoder) ReadByte() byte {

	b, err := f.r.ReadByte()
	if err != nil {
		panic(err)
	}
	return b

}

func (f StreamDecoder) ReadInt() (n int) {

	data, err := f.r.ReadNetUint32()
	if err != nil {
		panic(err)
	}
	n = int(data)

	return
}

func (f StreamDecoder) ReadShort() (n uint16) {

	data, err := f.r.ReadNetUint16()
	if err != nil {
		panic(err)
	}
	n = data

	return
}

func (f StreamDecoder) ReadLong() (n int64) {

	data, err := f.r.ReadNetUint64()
	if err != nil {
		panic(err)
	}
	n = int64(data)

	return
}

func (f StreamDecoder) ReadString() (s string) {
	size := f.ReadShort()

	str := make([]byte, size)
	_, err := f.r.Read(str)
	if err != nil {
		panic(err)
	}
	s = string(str)

	return
}

func (f StreamDecoder) ReadLongString() (s string) {

	size := f.ReadInt()

	str := make([]byte, size)
	_, err := f.r.Read(str)
	if err != nil {
		panic(err)
	}
	s = string(str)

	return
}

func (f StreamDecoder) ReadUUID() *UUID {

	bytes := make([]byte, 16)
	_, err := f.r.Read(bytes)
	if err != nil {
		panic(err)
	}
	u, _ := UUIDFromBytes(bytes)
	return &u

}

func (f StreamDecoder) ReadStringList() []string {
	size := f.ReadShort()

	l := make([]string, size)
	for i := 0; i < int(size); i++ {
		l[i] = f.ReadString()
	}

	return l
}

func (f StreamDecoder) ReadBytesInternal() []byte {
	size := f.ReadInt()
	if size < 0 {
		return nil
	}

	bytes := make([]byte, size)
	_, err := f.r.Read(bytes)
	if err != nil {
		panic(err)
	}
	return bytes

}

func (f StreamDecoder) ReadBytes() []byte {
	l := f.ReadBytesInternal()
	return l
}

func (f StreamDecoder) ReadShortBytes() []byte {
	size := f.ReadShort()

	bytes := make([]byte, size)
	_, err := f.r.Read(bytes)
	if err != nil {
		panic(err)
	}
	return bytes

}

func (f StreamDecoder) ReadInet() (net.IP, int) {

	size := f.ReadByte()
	if !(size == 4 || size == 16) {
		panic(fmt.Errorf("invalid IP size: %d", size))
	}

	ip := make([]byte, int(size))
	_, err := f.r.Read(ip)
	if err != nil {
		panic(err)
	}
	port := f.ReadInt()
	return net.IP(ip), port

}

func (f StreamDecoder) ReadConsistency() Consistency {
	return Consistency(f.ReadShort())
}

func (f StreamDecoder) ReadStringMap() map[string]string {
	size := f.ReadShort()
	m := make(map[string]string)

	for i := 0; i < int(size); i++ {
		k := f.ReadString()
		v := f.ReadString()
		m[k] = v
	}

	return m
}

func (f StreamDecoder) ReadBytesMap() map[string][]byte {
	size := f.ReadShort()
	m := make(map[string][]byte)

	for i := 0; i < int(size); i++ {
		k := f.ReadString()
		v := f.ReadBytes()
		m[k] = v
	}

	return m
}

func (f StreamDecoder) ReadStringMultiMap() map[string][]string {
	size := f.ReadShort()
	m := make(map[string][]string)

	for i := 0; i < int(size); i++ {
		k := f.ReadString()
		v := f.ReadStringList()
		m[k] = v
	}
	return m
}
