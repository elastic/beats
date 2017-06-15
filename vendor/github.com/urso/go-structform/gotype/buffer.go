package gotype

import (
	"sync"
	"unsafe"
)

var alignment = unsafe.Alignof((*uint32)(nil))

var stridePool = sync.Pool{}

type buffer struct {
	strides  []stride
	strides0 [8]stride
	i        int

	preAlloc uintptr
}

type stride struct {
	raw []byte
	pos uintptr
}

func (b *buffer) init(alloc int) {
	b.strides = b.strides0[:1]
	b.preAlloc = uintptr(alloc)
	b.i = 0

	s := &b.strides[0]
	s.raw = b.allocStride()
	s.pos = 0
}

func (b *buffer) allocStride() []byte {
	if bytesIfc := stridePool.Get(); bytesIfc != nil {
		return bytesIfc.([]byte)
	} else {
		return make([]byte, b.preAlloc)
	}
}

func (b *buffer) alloc(sz int) unsafe.Pointer {
	// align 'sz' to next for bytes after 'sz'
	aligned := (((uintptr(sz) + (alignment - 1)) / alignment) * alignment)
	total := aligned + alignment

	mem := b.doAlloc(total)

	szPtr := (*uint32)(unsafe.Pointer(&mem[aligned]))
	*szPtr = uint32(total)
	return unsafe.Pointer(&mem[0])
}

func (b *buffer) release() {
	s := &b.strides[b.i]
	if s.pos == 0 {
		panic("release of unallocated memory")
	}

	szPtr := (*uint32)(unsafe.Pointer(&s.raw[s.pos-alignment]))
	sz := uintptr(*szPtr)

	s.pos -= sz
	if s.pos == 0 && b.i > 0 {
		// release (last) stride
		stridePool.Put(s.raw)
		s.raw = nil
		b.strides = b.strides[:b.i]
		b.i--
	}
}

func (b *buffer) doAlloc(sz uintptr) []byte {
	s := &b.strides[b.i]
	space := uintptr(len(s.raw)) - s.pos

	if space < sz {
		var bytes []byte

		if b.preAlloc < sz {
			bytes = make([]byte, sz)
		} else {
			bytes = b.allocStride()
		}

		b.strides = append(b.strides, stride{
			raw: bytes,
			pos: sz,
		})
		b.i++

		return b.strides[b.i].raw[0:]
	}

	start := s.pos
	s.pos += sz

	mem := s.raw[start:s.pos]
	for i := range mem {
		mem[i] = 0
	}
	return mem
}
