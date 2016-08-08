package cassandra

import (
	"fmt"
	"io"
	"net"
)

type ByteArrayDecoder struct {
	Data []byte
}

func readInt(p []byte) int32 {
	return int32(p[0])<<24 | int32(p[1])<<16 | int32(p[2])<<8 | int32(p[3])
}

func (f *Framer) ReadHeader1() (head *frameHeader, err error) {
	p := make([]byte, 9)
	head = &frameHeader{}
	_, err = io.ReadFull(f.r, p[:1])
	if err != nil {
		return head, err
	}

	version := p[0] & protoVersionMask

	if version < protoVersion1 || version > protoVersion4 {
		return head, fmt.Errorf("unsupported response version: %d", version)
	}
	f.proto = version

	headSize := 9
	if version < protoVersion3 {
		headSize = 8
	}

	_, err = io.ReadFull(f.r, p[1:headSize])
	if err != nil {
		return head, err
	}

	p = p[:headSize]

	v := p[0]

	head.Version = protoVersion(v)

	head.Flags = p[1]

	if version > protoVersion2 {
		if len(p) != 9 {
			return head, fmt.Errorf("not enough bytes to read header require 9 got: %d", len(p))
		}

		head.Stream = int(int16(p[2])<<8 | int16(p[3]))
		head.Op = FrameOp(p[4])
		head.BodyLength = int(readInt(p[5:]))
	} else {
		if len(p) != 8 {
			return head, fmt.Errorf("not enough bytes to read header require 8 got: %d", len(p))
		}

		head.Stream = int(int8(p[2]))
		head.Op = FrameOp(p[3])
		head.BodyLength = int(readInt(p[4:]))
	}
	f.Header = head
	return head, nil
}

func (f ByteArrayDecoder) ReadByte() (b byte) {
	if len(f.Data) < 1 {
		panic(fmt.Errorf("not enough bytes in buffer to Read byte require 1 got: %d", len(f.Data)))
	}

	b = f.Data[0]
	f.Data = f.Data[1:]
	return
}

func (f ByteArrayDecoder) ReadInt() (n int) {
	if len(f.Data) < 4 {
		panic(fmt.Errorf("not enough bytes in buffer to Read int require 4 got: %d", len(f.Data)))
	}

	n = int(int32(f.Data[0])<<24 | int32(f.Data[1])<<16 | int32(f.Data[2])<<8 | int32(f.Data[3]))
	f.Data = f.Data[4:]

	return
}

func (f ByteArrayDecoder) ReadShort() (n uint16) {
	if len(f.Data) < 2 {
		panic(fmt.Errorf("not enough bytes in buffer to Read short require 2 got: %d", len(f.Data)))
	}
	n = uint16(f.Data[0])<<8 | uint16(f.Data[1])
	f.Data = f.Data[2:]
	return
}

func (f ByteArrayDecoder) ReadLong() (n int64) {
	if len(f.Data) < 8 {
		panic(fmt.Errorf("not enough bytes in buffer to Read long require 8 got: %d", len(f.Data)))
	}
	n = int64(f.Data[0])<<56 | int64(f.Data[1])<<48 | int64(f.Data[2])<<40 | int64(f.Data[3])<<32 |
		int64(f.Data[4])<<24 | int64(f.Data[5])<<16 | int64(f.Data[6])<<8 | int64(f.Data[7])
	f.Data = f.Data[8:]
	return
}

func (f ByteArrayDecoder) ReadString() (s string) {
	size := f.ReadShort()

	if len(f.Data) < int(size) {
		panic(fmt.Errorf("not enough bytes in buffer to Read string require %d got: %d", size, len(f.Data)))
	}

	s = string(f.Data[:size])
	f.Data = f.Data[size:]
	return
}

func (f ByteArrayDecoder) ReadLongString() (s string) {
	size := f.ReadInt()

	if len(f.Data) < size {
		panic(fmt.Errorf("not enough bytes in buffer to Read long string require %d got: %d", size, len(f.Data)))
	}

	s = string(f.Data[:size])
	f.Data = f.Data[size:]
	return
}

func (f ByteArrayDecoder) ReadUUID() *UUID {
	if len(f.Data) < 16 {
		panic(fmt.Errorf("not enough bytes in buffer to Read uuid require %d got: %d", 16, len(f.Data)))
	}

	u, _ := UUIDFromBytes(f.Data[:16])
	f.Data = f.Data[16:]
	return &u
}

func (f ByteArrayDecoder) ReadStringList() []string {
	size := f.ReadShort()

	l := make([]string, size)
	for i := 0; i < int(size); i++ {
		l[i] = f.ReadString()
	}

	return l
}

func (f ByteArrayDecoder) ReadBytesInternal() []byte {
	size := f.ReadInt()
	if size < 0 {
		return nil
	}

	if len(f.Data) < size {
		panic(fmt.Errorf("not enough bytes in buffer to Read bytes require %d got: %d", size, len(f.Data)))
	}

	l := f.Data[:size]
	f.Data = f.Data[size:]

	return l
}

func (f ByteArrayDecoder) ReadBytes() []byte {
	l := f.ReadBytesInternal()

	return l
}

func (f ByteArrayDecoder) ReadShortBytes() []byte {
	size := f.ReadShort()
	if len(f.Data) < int(size) {
		panic(fmt.Errorf("not enough bytes in buffer to Read short bytes: require %d got %d", size, len(f.Data)))
	}

	l := f.Data[:size]
	f.Data = f.Data[size:]

	return l
}

func (f ByteArrayDecoder) ReadInet() (net.IP, int) {
	if len(f.Data) < 1 {
		panic(fmt.Errorf("not enough bytes in buffer to Read inet size require %d got: %d", 1, len(f.Data)))
	}

	size := f.Data[0]
	f.Data = f.Data[1:]

	if !(size == 4 || size == 16) {
		panic(fmt.Errorf("invalid IP size: %d", size))
	}

	if len(f.Data) < 1 {
		panic(fmt.Errorf("not enough bytes in buffer to Read inet require %d got: %d", size, len(f.Data)))
	}

	ip := make([]byte, size)
	copy(ip, f.Data[:size])
	f.Data = f.Data[size:]

	port := f.ReadInt()
	return net.IP(ip), port
}

func (f ByteArrayDecoder) ReadConsistency() Consistency {
	return Consistency(f.ReadShort())
}

func (f ByteArrayDecoder) ReadStringMap() map[string]string {
	size := f.ReadShort()
	m := make(map[string]string)

	for i := 0; i < int(size); i++ {
		k := f.ReadString()
		v := f.ReadString()
		m[k] = v
	}

	return m
}

func (f ByteArrayDecoder) ReadBytesMap() map[string][]byte {
	size := f.ReadShort()
	m := make(map[string][]byte)

	for i := 0; i < int(size); i++ {
		k := f.ReadString()
		v := f.ReadBytes()
		m[k] = v
	}

	return m
}

func (f ByteArrayDecoder) ReadStringMultiMap() map[string][]string {
	size := f.ReadShort()
	m := make(map[string][]string)

	for i := 0; i < int(size); i++ {
		k := f.ReadString()
		v := f.ReadStringList()
		m[k] = v
	}
	return m
}
