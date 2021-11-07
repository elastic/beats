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
	"unsafe"

	bin "github.com/urso/go-bin"

	"github.com/elastic/go-txfile"
)

// primitive types
type (
	u32 = bin.U32le
	u64 = bin.U64le
)

// queuePage is the root structure into a persisted Queue instance.
type queuePage struct {
	version u32

	// start/end of single linked events list and event ids
	head pos // head points to first event in list
	tail pos // tail points to next event to be written

	// read points to next event to continue reading from
	// if read == tail, all events have been read
	read pos

	inuse u64 // number of actively used data pages
}

type pos struct {
	offset u64 // file offset of event
	id     u64 // id of event
}

// eventPage create a single list of event pages, storing a number
// of events per page.
// If off == 0, the page does contain data only.
type eventPage struct {
	next  u64 // PageID of next eventPage
	first u64 // event id of first event in current page
	last  u64 // event id of last even in current page
	off   u32 // offset of first event in current page
}

// eventHeader is keeps track of the event size in bytes.
// The event ID can be 'computed' by iterating the events in a page.
type eventHeader struct {
	sz u32
}

const (
	queueVersion = 1

	// SzRoot is the size of the queue header in bytes.
	SzRoot = int(unsafe.Sizeof(queuePage{}))

	szEventPageHeader = int(unsafe.Sizeof(eventPage{}))
	szEventHeader     = int(unsafe.Sizeof(eventHeader{}))
)

func castQueueRootPage(b []byte) (hdr *queuePage) { bin.UnsafeCastStruct(&hdr, b); return }

func castEventPageHeader(b []byte) (hdr *eventPage) { bin.UnsafeCastStruct(&hdr, b); return }

func castEventHeader(b []byte) (hdr *eventHeader) { bin.UnsafeCastStruct(&hdr, b); return }

func traceQueueHeader(hdr *queuePage) {
	traceln("queue header:")
	traceln("  version:", hdr.version.Get())
	tracef("  head(%v, %v)\n", hdr.head.id.Get(), hdr.head.offset.Get())
	tracef("  tail(%v, %v)\n", hdr.tail.id.Get(), hdr.tail.offset.Get())
	tracef("  read(%v, %v)\n", hdr.read.id.Get(), hdr.read.offset.Get())
	traceln("  data pages", hdr.inuse.Get())
}

func tracePageHeader(id txfile.PageID, hdr *eventPage) {
	tracef("event page %v (next=%v, first=%v, last=%v, off=%v)\n",
		id, hdr.next.Get(), hdr.first.Get(), hdr.last.Get(), hdr.off.Get())
}
