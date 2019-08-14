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

import "github.com/elastic/go-txfile"

// Access provides transaction support and access to pages and queue header.
// It wraps the Delegate for providing a common interface for working with
// transactions and files.
type access struct {
	Delegate
	rootID  txfile.PageID
	rootOff int

	quID queueID
}

func makeAccess(delegate Delegate) (access, ErrKind) {
	rootID, rootOff := delegate.Root()
	if rootID == 0 {
		return access{}, NoQueueRoot
	}

	return access{
		Delegate: delegate,
		rootID:   rootID,
		rootOff:  int(rootOff),
	}, NoError
}

// ReadRoot reads the root page into an array.
// ReadRoot create a short lived read transaction for accessing and copying the
// queue root.
func (a *access) ReadRoot() ([SzRoot]byte, reason) {
	const op = "pq/read-queue-root"

	var buf [SzRoot]byte

	tx, err := a.BeginRead()
	if err != nil {
		return buf, a.errWrap(op, err)
	}
	defer tx.Close()

	fail := NoError
	err = withPage(tx, a.rootID, func(page []byte) {
		n := copy(buf[:], page[a.rootOff:])
		if n < SzRoot {
			fail = InvalidQueueRoot
		}
	})

	if err != nil {
		return buf, a.errWrap(op, err)
	}
	if fail != NoError {
		return buf, a.err(op).of(fail)
	}

	return buf, nil
}

// rootPage accesses the queue root page from within the passed transaction.
func (a *access) rootPage(tx *txfile.Tx) (*txfile.Page, error) {
	return tx.Page(a.rootID)
}

func (a *access) RootFileOffset() uintptr {
	return a.Offset(a.rootID, uintptr(a.rootOff))
}

// LoadRootPage accesses the queue root page from within the passed write
// transaction.
// The Root page it's content is loaded into the write buffer for manipulations.
// The page returned is not marked as dirty yet.
func (a *access) LoadRootPage(tx *txfile.Tx) (*txfile.Page, *queuePage, reason) {
	const op = "pq/load-queue-root"

	var hdr *queuePage
	page, err := a.rootPage(tx)
	if err == nil {
		err = page.Load()
		if err == nil {
			buf, _ := page.Bytes()
			hdr = castQueueRootPage(buf[a.rootOff:])
		}
	}

	if err != nil {
		msg := "Error reading the queue header"
		return nil, nil, a.errWrap(op, err).of(ReadFail).report(msg)
	}
	return page, hdr, nil
}

// RootHdr returns a pointer to the queue root header. The pointer to the
// header is only valid as long as the transaction is still active.
func (a *access) RootHdr(tx *txfile.Tx) (*queuePage, reason) {
	const op = "pq/read-queue-header"

	var hdr *queuePage
	err := withPage(tx, a.rootID, func(buf []byte) {
		hdr = castQueueRootPage(buf[a.rootOff:])
	})
	if err != nil {
		msg := "Error reading the queue header"
		return nil, a.errWrap(op, err).of(ReadFail).report(msg)
	}

	return hdr, nil
}

// ParsePosition parses an on disk position, providing page id, page offset and
// event id in a more accessible format.
func (a *access) ParsePosition(p *pos) position {
	page, off := a.SplitOffset(uintptr(p.offset.Get()))
	if page != 0 && off == 0 {
		off = uintptr(a.PageSize())
	}

	return position{
		page: page,
		off:  int(off),
		id:   p.id.Get(),
	}
}

// WritePosition serializes a position into it's on-disk representation.
func (a *access) WritePosition(to *pos, pos position) {
	pageOff := pos.off
	if pageOff == a.PageSize() {
		pageOff = 0 // use 0 to mark page offset as end-of-page
	}

	off := a.Offset(pos.page, uintptr(pageOff))
	to.offset.Set(uint64(off))
	to.id.Set(pos.id)
}

func (a *access) readPageByID(pool *pagePool, id txfile.PageID) (*page, reason) {
	const op = "pq/read-single-page"

	tx, err := a.BeginRead()
	if err != nil {
		return nil, a.errWrap(op, err)
	}

	defer tx.Close()

	var page *page
	err = withPage(tx, id, func(buf []byte) {
		page = pool.NewPageWith(id, buf)
	})
	if err != nil {
		return nil, a.errWrapPage(op, id, err).of(ReadFail)
	}

	return page, nil
}

func (a *access) err(op string) *Error { return a.errPage(op, 0) }
func (a *access) errPage(op string, id txfile.PageID) *Error {
	return &Error{op: op, ctx: a.errPageCtx(id)}
}

func (a *access) errWrap(op string, cause error) *Error { return a.errWrapPage(op, 0, cause) }
func (a *access) errWrapPage(op string, id txfile.PageID, cause error) *Error {
	return a.errPage(op, id).causedBy(cause)
}

func (a *access) errCtx() errorCtx { return errorCtx{id: a.quID} }
func (a *access) errPageCtx(id txfile.PageID) errorCtx {
	if id != 0 {
		return errorCtx{id: a.quID, isPage: true, page: id}
	}
	return errorCtx{id: a.quID}
}
