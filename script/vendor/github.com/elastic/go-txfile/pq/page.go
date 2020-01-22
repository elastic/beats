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

package pq

import (
	"sync"

	"github.com/elastic/go-txfile"
)

// page is used by the write buffer to keep page content and on-disk
// assignment. Pages with Meta.ID == 0 are not allocated on disk yet.
type page struct {
	Next *page

	Meta pageMeta
	Data []byte
}

type pageMeta struct {
	ID              txfile.PageID
	FirstID, LastID uint64
	FirstOff        uint32
	EndOff          uint32
	Flags           pageFlags
}

type pageFlags struct {
	Dirty bool // indicates new event contents being written to this page
}

type pagePool struct {
	sync.Pool
}

func newPagePool(pageSize int) *pagePool {
	return &pagePool{sync.Pool{
		New: func() interface{} {
			return &page{
				Data: make([]byte, pageSize),
			}
		},
	}}
}

func (pp *pagePool) NewPage() *page {
	return pp.get()
}

func (pp *pagePool) NewPageWith(id txfile.PageID, contents []byte) *page {
	p := pp.NewPage()
	copy(p.Data, contents)
	hdr := castEventPageHeader(contents)
	p.Meta = pageMeta{
		ID:       id,
		FirstID:  hdr.first.Get(),
		LastID:   hdr.last.Get(),
		FirstOff: hdr.off.Get(),
	}
	return p
}

func (pp *pagePool) get() *page { return pp.Pool.Get().(*page) }

func (pp *pagePool) Release(p *page) {
	p.Clear()
	pp.Pool.Put(p)
}

// Clear zeroes out a page object and the buffer page header, preparing the
// page object for being reused.
func (p *page) Clear() {
	p.Meta = pageMeta{}
	p.Next = nil

	// clear page header
	for i := 0; i < szEventPageHeader; i++ {
		p.Data[i] = 0
	}
}

// Assigned checks if the page is represented by on on-disk page.
func (p *page) Assigned() bool {
	return p.Meta.ID != 0
}

// Dirty checks if the page is dirty and must be flushed.
func (p *page) Dirty() bool {
	return p.Meta.Flags.Dirty
}

// MarkDirty marks a page as dirty.
func (p *page) MarkDirty() {
	p.Meta.Flags.Dirty = true
}

// UnmarkDirty marks a page as being in sync with the on-disk page.
func (p *page) UnmarkDirty() {
	p.Meta.Flags.Dirty = false
}

// SetNext write the next page ID into the page header.
func (p *page) SetNext(id txfile.PageID) {
	hdr := castEventPageHeader(p.Data)
	hdr.next.Set(uint64(id))
}

// Payload returns the slice of the page it's complete payload.
func (p *page) Payload() []byte {
	return p.Data[szEventPageHeader:]
}

// UpdateHeader updates the page header to reflect the page meta-data pages.
func (p *page) UpdateHeader() {
	hdr := castEventPageHeader(p.Data)
	hdr.first.Set(p.Meta.FirstID)
	hdr.last.Set(p.Meta.LastID)
	hdr.off.Set(p.Meta.FirstOff)
}
