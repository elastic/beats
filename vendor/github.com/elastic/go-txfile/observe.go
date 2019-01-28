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

import "time"

// Observer defines common callbacks to observe errors, transactions and other
// state changes in txfile. The callbacks must not block, so to not block any
// file operations.
type Observer interface {

	// OnOpen reports initial file stats when successfully open a file.
	//
	// Memory stats are reported in sizes. Page counts can be derived by dividing
	// the sizes by pageSz.
	//
	// Derived metrics:
	//   dataAreaSz = maxSz - metaAreaSz       // total data area size
	//   dataAreaActive = dataAreaSz - avail   // data area bytes currently in use
	OnOpen(stats FileStats)

	// OnBegin reports the start of a new transaction.
	OnTxBegin(readonly bool)

	// OnClose is used to signal the end of a transaction.
	//
	// If readonly is set, the transaction we a readonly transaction. Only the
	// Duration, Total, and Accessed fields will be set.
	// Only if `commit` is set will the reported stats be affective in upcoming
	// file operations (pages written/freed).
	OnTxClose(file FileStats, tx TxStats)
}

// FileStats reports the current file state like version and allocated/free space.
type FileStats struct {
	Version       uint32 // lates file-header version
	Size          uint64 // actual file size (changes if file did grow dynamically due to allocations)
	MaxSize       uint64 // max file size as stores in file header
	PageSize      uint32 // file page size
	MetaArea      uint   // total pages reserved for the meta area
	DataAllocated uint   // data pages in use
	MetaAllocated uint   // meta pages in use
}

// TxStats contains common statistics collected during the life-cycle of a transaction.
type TxStats struct {
	Readonly  bool          // set if transaction is readonly. In this case only Duration, Total and Accessed will be set.
	Commit    bool          // If set reported stats will be affective in future file operations. Otherwise allocation stats will have no effect.
	Duration  time.Duration // total duration the transaction was live
	Total     uint          // total number of pages accessed(written, read, changed) during the transaction
	Accessed  uint          // number of accessed existing pages (read)
	Allocated uint          // temporarily allocated pages
	Freed     uint          // total number of freed pages
	Written   uint          // total number of pages being written to
	Updated   uint          // number of pages with changed contents
}
