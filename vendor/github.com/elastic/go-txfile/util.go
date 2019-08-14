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
	"math/bits"
)

// pagingWriter supports writing entries into a linked (pre-allocated) list of
// pages.
type pagingWriter struct {
	ids         idList
	buf         []byte
	pageSize    uint
	extraHeader uint

	onPage func(id PageID, buf []byte) reason

	// current page state
	i       int
	off     uint
	hdr     *listPage
	page    []byte
	payload []byte
	count   uint32
}

const maxUint uint = ^uint(0)

func newPagingWriter(
	ids idList,
	pageSize uint,
	extraHeader uint,
	onPage func(id PageID, buf []byte) reason,
) *pagingWriter {
	if len(ids) == 0 {
		return nil
	}

	buf := make([]byte, len(ids)*int(pageSize))

	// prelink all pages, in case some are not written to
	off := 0
	for _, id := range ids[1:] {
		hdr, _ := castListPage(buf[off:])
		hdr.next.Set(id)
		off += int(pageSize)
	}

	w := &pagingWriter{
		ids:         ids,
		buf:         buf,
		pageSize:    pageSize,
		extraHeader: extraHeader,
		onPage:      onPage,
	}
	w.prepareNext("")
	return w
}

func (w *pagingWriter) Write(entry []byte) reason {
	const op = "txfile/write-meta-list"

	if w == nil {
		return nil
	}

	if len(w.payload) < len(entry) {
		if err := w.flushCurrent(op); err != nil {
			return err
		}
	}

	n := copy(w.payload, entry)
	w.payload = w.payload[n:]
	w.count++
	return nil
}

func (w *pagingWriter) Flush() reason {
	const op = "txfile/flush-meta-list"

	if w == nil {
		return nil
	}

	if err := w.finalizePage(); err != nil {
		return err
	}

	for w.i < len(w.ids) {
		// update to next page
		w.prepareNext(op)
		if err := w.finalizePage(); err != nil {
			return err
		}
	}

	return nil
}

func (w *pagingWriter) flushCurrent(op string) (err reason) {
	if err = w.finalizePage(); err == nil {
		err = w.prepareNext(op)
	}
	return
}

func (w *pagingWriter) finalizePage() reason {
	w.hdr.count.Set(w.count)
	if w.onPage != nil {
		if err := w.onPage(w.ids[w.i], w.page); err != nil {
			return err
		}
	}

	w.count = 0
	w.off += w.pageSize
	w.i++
	return nil
}

func (w *pagingWriter) prepareNext(op string) reason {
	if w.i >= len(w.ids) {
		return errOp(op).of(InternalError).report("Not enough pages pre-allocated")
	}

	w.page = w.buf[w.off : w.off+w.pageSize]
	w.hdr, w.payload = castListPage(w.page)
	w.payload = w.payload[w.extraHeader:]
	return nil
}

func isPowerOf2(v uint64) bool {
	// an uint is a power of two if exactly one bit is set ->
	return v > 0 && (v&(v-1)) == 0
}

// nextPowerOf2 computes the next power of two value of `u`, such that
// nextPowerOf2(u) > u
// The input value must not have the highest bit being set.
func nextPowerOf2(u uint64) uint64 {
	b := uint64(bits.LeadingZeros64(u))
	return uint64(1) << (64 - b)
}

func ignoreReason(fn func() reason) func() {
	return func() { _ = fn() }
}
