package nfs

import (
	"encoding/binary"
)

type Xdr struct {
	data   []byte
	offset uint32
}

func (xdr *Xdr) size() int {
	return len(xdr.data)
}

func (xdr *Xdr) getInt() int32 {
	i := int32(binary.BigEndian.Uint32(xdr.data[xdr.offset : xdr.offset+4]))
	xdr.offset += 4
	return int32(i)
}

func (xdr *Xdr) getUInt() uint32 {
	i := uint32(binary.BigEndian.Uint32(xdr.data[xdr.offset : xdr.offset+4]))
	xdr.offset += 4
	return i
}

func (xdr *Xdr) getHyper() int64 {
	i := int64(binary.BigEndian.Uint64(xdr.data[xdr.offset : xdr.offset+8]))
	xdr.offset += 8
	return i
}

func (xdr *Xdr) getUHyper() uint64 {
	i := uint64(binary.BigEndian.Uint64(xdr.data[xdr.offset : xdr.offset+8]))
	xdr.offset += 8
	return i
}

func (xdr *Xdr) getString() string {
	return string(xdr.getDynamicOpaque())
}

func (xdr *Xdr) getOpaque(length uint32) []byte {
	padding := (4 - (length & 3)) & 3
	b := xdr.data[xdr.offset : xdr.offset+length]
	xdr.offset += length + padding
	return b
}

func (xdr *Xdr) getDynamicOpaque() []byte {
	l := xdr.getUInt()
	return xdr.getOpaque(l)
}

func (xdr *Xdr) getUIntVector() []uint32 {
	l := xdr.getUInt()
	v := make([]uint32, int(l))
	for i := 0; i < len(v); i++ {
		v[i] = xdr.getUInt()
	}
	return v
}
