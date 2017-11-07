package cassandra

import (
	"net"
)

type Decoder interface {
	ReadByte() (byte, error)

	ReadInt() (n int)

	ReadShort() (n uint16)

	ReadLong() (n int64)

	ReadString() (s string)

	ReadLongString() (s string)

	ReadUUID() *UUID

	ReadStringList() []string

	ReadBytesInternal() []byte

	ReadBytes() []byte

	ReadShortBytes() []byte

	ReadInet() (net.IP, int)

	ReadConsistency() Consistency

	ReadStringMap() map[string]string

	ReadBytesMap() map[string][]byte

	ReadStringMultiMap() map[string][]string
}
