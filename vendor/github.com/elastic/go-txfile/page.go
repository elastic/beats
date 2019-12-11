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

// Page provides access to an on disk page.
// Pages can only be overwritten from within a read-write Transaction.
// Writes are be buffered until transaction commit, such that other but the
// current transaction will not be able to see file changes.
type Page struct {
	id       PageID // Original PageID for user access.
	ondiskID PageID // On disk PageID. If contents is loaded from overwrite page, ondiskID != id

	tx    *Tx    // Parent transaction.
	bytes []byte // Page contents.
	flags pageFlags
}

// PageID used to reference a file pages.
type PageID uint64

type pageFlags struct {
	new     bool // page has been allocated. No on-disk contents.
	freed   bool // page has been freed within current transaction.
	flushed bool // page has already been flushed. No more writing possible.
	cached  bool // original page contents is copied in memory and can be overwritten.
	dirty   bool // page is marked as dirty and will be written on commit
}

const minPageSize = 1024

// newPage creates a new page context within the current transaction.
func newPage(tx *Tx, id PageID) *Page {
	return &Page{id: id, ondiskID: id, tx: tx}
}

func (p *Page) onWriteNew() {
	p.tx.accessStats.New++
}

func (p *Page) onUpdated() {
	p.tx.accessStats.Update++
}

// ID returns the pages PageID. The ID can be used to store a reference
// to this page, for use within another transaction.
func (p *Page) ID() PageID { return p.id }

// Readonly checks if the page is accessed in readonly mode.
func (p *Page) Readonly() bool { return p.tx.Readonly() }

// Writable checks if the page can be written to.
func (p *Page) Writable() bool { return !p.Readonly() }

// Dirty reports if the page is marked as dirty and needs to be flushed on
// commit.
func (p *Page) Dirty() bool { return p.flags.dirty }

// MarkDirty marks a page as dirty. MarkDirty should only be used if
// in-place modification to the pages buffer have been made, after use of Load().
func (p *Page) MarkDirty() error {
	const op = "txfile/page-mark-dirty"

	if err := p.canWrite(op); err != nil {
		return err
	}

	p.setDirty()
	return nil
}

func (p *Page) setDirty() {
	if p.flags.dirty {
		return
	}

	p.flags.dirty = true
	if p.flags.new {
		p.onWriteNew()
	} else {
		p.onUpdated()
	}
}

// Free marks a page as free. Freeing a dirty page will return an error.
// The page will be returned to the allocator when the transaction commits.
func (p *Page) Free() error {
	const op = "txfile/page-free"

	if err := p.canWrite(op); err != nil {
		return err
	}
	if p.flags.dirty {
		const msg = "freeing dirty pages is not allowed"
		return &Error{op: op, kind: InvalidOp, ctx: p.errCtx(), msg: msg}
	}

	p.tx.freePage(p.id)
	if p.id != p.ondiskID {
		p.tx.freeWALID(p.id, p.ondiskID)
	}

	p.flags.freed = true
	return nil
}

// Bytes returns the page its contents.
// One can only modify the buffer in write transaction, if Load() or SetBytes()
// have been called before Bytes(). Otherwise a non-recoverable BUS panic might
// be triggerd (program will be killed by OS).
// Bytes returns an error if the page has just been allocated (no backing buffer)
// or the transaction is already been closed.
// Use SetBytes() or Load(), to initialize the buffer of a newly allocated page.
func (p *Page) Bytes() ([]byte, error) {
	const op = "txfile/page-bytes"

	if err := p.canRead(op); err != nil {
		return nil, err
	}
	if p.bytes == nil && p.flags.new {
		const msg = "can not read contents of fresh allocated page without contents"
		return nil, &Error{op: op, kind: InvalidOp, ctx: p.errCtx(), msg: msg}
	}

	return p.getBytes(op)
}

func (p *Page) getBytes(op string) ([]byte, reason) {
	if p.bytes == nil {
		bytes := p.tx.access(p.ondiskID)
		if bytes == nil {
			cause := raiseOutOfBounds(p.ondiskID)
			return nil, &Error{op: op, ctx: p.errCtx(), cause: cause}
		}

		p.bytes = bytes
	}

	return p.bytes, nil
}

// Load reads the pages original contents into a cached memory buffer, allowing
// for in-place modifications to the page. Load returns and error, if used from
// within a readonly transaction.
// If the page has been allocated from within the current transaction, a new
// temporary buffer will be allocated.
// After load, the write-buffer can be accessed via Bytes(). After modifications to the buffer,
// one must use MarkDirty(), so the page will be flushed on commit.
func (p *Page) Load() error {
	const op = "txfile/page-load-writable"

	if err := p.canWrite(op); err != nil {
		return err
	}

	return p.loadBytes(op)
}

func (p *Page) loadBytes(op string) reason {
	if p.flags.cached {
		return nil
	}

	if p.flags.new {
		p.flags.cached = true
		p.bytes = make([]byte, p.tx.PageSize())
		return nil
	}

	if p.flags.dirty {
		p.flags.cached = true
		return nil
	}

	// copy original contents into writable buffer (page needs to be marked dirty if contents is overwritten)
	orig, err := p.getBytes(op)
	if err != nil {
		return err
	}
	tmp := make([]byte, len(orig))
	copy(tmp, orig)
	p.bytes = tmp
	p.flags.cached = true

	return nil
}

// SetBytes sets the new contents of a page. If the size of contents is less
// then the files page size, the original contents must be read.  If the length
// of contents matches the page size, a reference to the contents buffer will
// be held. To enforce a copy, use Load(), Bytes(), copy() and MarkDirty().
func (p *Page) SetBytes(contents []byte) error {
	const op = "txfile/page-set-bytes"

	if err := p.canWrite(op); err != nil {
		return err
	}

	pageSize := p.tx.PageSize()
	if len(contents) > pageSize {
		const msg = "page contents must not exceed the page size"
		return &Error{op: op, kind: InvalidParam, ctx: p.errCtx(), msg: msg}
	}

	if len(contents) < pageSize {
		if err := p.loadBytes(op); err != nil {
			return err
		}
		copy(p.bytes, contents)
	} else {
		p.bytes = contents
	}

	p.setDirty()
	return nil
}

// Flush flushes the page write buffer, if the page is marked as dirty.
// The page its contents must not be changed after calling Flush, as the flush
// is executed asynchronously in the background.
// Dirty pages will be automatically flushed on commit.
func (p *Page) Flush() error {
	const op = "txfile/page-flush"

	if err := p.canWrite(op); err != nil {
		return err
	}

	return p.doFlush(op)
}

func (p *Page) doFlush(op string) reason {
	if !p.flags.dirty || p.flags.flushed {
		return nil
	}

	if !p.flags.new {
		if p.id == p.ondiskID {
			walID := p.tx.allocWALID(p.id)
			if walID == 0 {
				const msg = "not enough space to allocate write ahead page"
				return &Error{op: op, kind: OutOfMemory, ctx: p.errCtx(), msg: msg}
			}
			p.ondiskID = walID
		} else {
			// page already in WAL -> free WAL page and write into original page
			p.tx.freeWALID(p.id, p.ondiskID)
			p.ondiskID = p.id
		}
	}

	p.flags.flushed = true
	p.tx.scheduleWrite(p.ondiskID, p.bytes)
	return nil
}

func (p *Page) canRead(op string) *Error {
	err := p.tx.canRead(op)
	if err != nil {
		err.ctx = p.errCtx()
	}
	return err
}

func (p *Page) canWrite(op string) *Error {
	if err := p.tx.canWrite(op); err != nil {
		err.ctx = p.errCtx()
		return err
	}

	var msg string
	switch {
	case p.flags.freed:
		msg = "page is already freed"
	case p.flags.flushed:
		msg = "page is already flushed"
	}

	if msg != "" {
		return &Error{op: op, kind: InvalidOp, ctx: p.errCtx(), msg: msg}
	}
	return nil
}

func (p *Page) errCtx() errorCtx {
	ctx := p.tx.errCtx()
	ctx.page, ctx.isPage = p.id, true
	return ctx
}
