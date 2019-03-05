// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package txfile

import (
	"fmt"
	"hash/fnv"
	"reflect"
	"unsafe"

	bin "github.com/urso/go-bin"
)

// on disk page layout for writing and parsing

// primitive types:
type (
	u8  = bin.U8le
	u16 = bin.U16le
	u32 = bin.U32le
	u64 = bin.U64le
	i8  = bin.I8le
	i16 = bin.I16le
	i32 = bin.I32le
	i64 = bin.I64le

	pgID u64
)

// Special page at beginning of file.
// A file holds to meta pages at the beginning of the file. A metaPage is
// updated after a write transaction has been completed.  On error during
// transactions or when updating the metaPage, the old metaPage will still be
// valid, technically ignoring all contents written by the transactions active
// while the program/id did crash/fail.
type metaPage struct {
	magic         u32
	version       u32
	pageSize      u32
	maxSize       u64 // maximum file size
	flags         u32
	root          pgID // ID of first page to look for data.
	txid          u64  // page transaction ID
	freelist      pgID // pointer to user area freelist
	wal           pgID // write-ahead-log root
	dataEndMarker pgID // end marker of user-area page
	metaEndMarker pgID // file end marker
	metaTotal     u64  // total number of pages in meta area
	checksum      u32
}

type metaBuf [unsafe.Sizeof(metaPage{})]byte

const (
	metaFlagPrealloc = 1 << 0 // indicates the complete file has been preallocated
)

type listPage struct {
	next  pgID // pointer to next entry
	count u32  // number of entries in current page
}

type freePage = listPage
type walPage = listPage

const (
	metaPageHeaderSize = int(unsafe.Sizeof(metaPage{}))
	listPageHeaderSize = int(unsafe.Sizeof(listPage{}))
	walPageHeaderSize  = int(unsafe.Sizeof(walPage{}))
	freePageHeaderSize = int(unsafe.Sizeof(freePage{}))
)

const magic uint32 = 0xBEA77AEB
const version uint32 = 1

func init() {
	checkPacked := func(t reflect.Type) {
		off := uintptr(0)
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.Offset != off {
				panic(fmt.Sprintf("field %v offset mismatch (expected=%v, actual=%v)",
					f.Name, off, f.Offset))
			}
			off += f.Type.Size()
		}
	}

	// check compiler really generates packed structes. Required, so file can be
	// accesed from within different architectures + checksum based on raw bytes
	// contents are correct.
	checkPacked(reflect.TypeOf(metaPage{}))
	checkPacked(reflect.TypeOf(freePage{}))
	checkPacked(reflect.TypeOf(walPage{}))
}

func castMetaPage(b []byte) (p *metaPage) { castPageTo(&p, b); return }

func (m *metaPage) Init(flags uint32, pageSize uint32, maxSize uint64) {
	m.magic.Set(magic)
	m.version.Set(version)
	m.pageSize.Set(pageSize)
	m.maxSize.Set(maxSize)
	m.flags.Set(flags)
	m.root.Set(0)
	m.freelist.Set(0)
	m.wal.Set(0)
	m.dataEndMarker.Set(0)
}

func (m *metaPage) Finalize() {
	m.checksum.Set(m.computeChecksum())
}

func (m *metaPage) Validate() reason {
	if m.magic.Get() != magic {
		return errOf(InvalidMetaPage).report("invalid magic number")
	}
	if m.version.Get() != version {
		return errOf(InvalidMetaPage).report("invalid version number")
	}
	if m.checksum.Get() != m.computeChecksum() {
		return errOf(InvalidMetaPage).report("checksum mismatch")
	}

	return nil
}

func (b *metaBuf) cast() *metaPage { return castMetaPage((*b)[:]) }

func (m *metaPage) computeChecksum() uint32 {
	h := fnv.New32a()
	type metaHashContent [unsafe.Offsetof(metaPage{}.checksum)]byte
	contents := *(*metaHashContent)(unsafe.Pointer(m))
	_, _ = h.Write(contents[:])
	return h.Sum32()
}

func (id *pgID) Len() int     { return id.access().Len() }
func (id *pgID) Get() PageID  { return PageID(id.access().Get()) }
func (id *pgID) Set(v PageID) { id.access().Set(uint64(v)) }
func (id *pgID) access() *u64 { return (*u64)(id) }

func castU8(b []byte) (u *u8)   { mapMem(&u, b); return }
func castU16(b []byte) (u *u16) { mapMem(&u, b); return }
func castU32(b []byte) (u *u32) { mapMem(&u, b); return }
func castU64(b []byte) (u *u64) { mapMem(&u, b); return }

func castListPage(b []byte) (node *listPage, data []byte) {
	if castPageTo(&node, b); node != nil {
		data = b[unsafe.Sizeof(listPage{}):]
	}
	return
}

func castFreePage(b []byte) (node *freePage, data []byte) {
	return castListPage(b)
}

func castWalPage(b []byte) (node *walPage, data []byte) {
	return castListPage(b)
}

func mapMem(to interface{}, b []byte) {
	bin.UnsafeCastStruct(to, b)
}

func castPageTo(to interface{}, b []byte) {
	mapMem(to, b)
}

func traceMetaPage(meta *metaPage) {
	traceln("meta page:")
	traceln("    version:", meta.version.Get())
	traceln("    pagesize:", meta.pageSize.Get())
	traceln("    maxsize:", meta.maxSize.Get())
	traceln("    root:", meta.root.Get())
	traceln("    txid:", meta.txid.Get())
	traceln("    freelist:", meta.freelist.Get())
	traceln("    wal:", meta.wal.Get())
	traceln("    data end:", meta.dataEndMarker.Get())
	traceln("    meta end:", meta.metaEndMarker.Get())
	traceln("    meta total:", meta.metaTotal.Get())
	traceln("    checksum:", meta.checksum.Get())
}
