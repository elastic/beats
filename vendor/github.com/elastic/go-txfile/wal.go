package txfile

import "unsafe"

// waLog (write-ahead-log) mapping page ids to overwrite page ids in
// the write-ahead-log.
type waLog struct {
	mapping   walMapping
	metaPages regionList
}

type txWalState struct {
	free     pageSet    // ids being freed
	new      walMapping // all wal pages used for overwrites in a transaction
	walLimit uint       // transaction wal page count -> execute checkpoint when reached
}

// walCommitState keeps track of changes applied to the wal log during the
// commit. These changes must be recorded for now, as the new wal state must
// not be updated in memory until after the transaction has been commit to disk.
type walCommitState struct {
	tx           *txWalState
	updated      bool
	checkpoint   bool
	mapping      walMapping // new wal mapping
	allocRegions regionList // pre-allocate meta pages for serializing new mapping
}

type walMapping map[PageID]PageID

const (
	walHeaderSize = uint(unsafe.Sizeof(walPage{}))
	walEntrySize  = 14

	defaultWALLimit = 1000
)

func makeWALog() waLog {
	return waLog{
		mapping:   walMapping{},
		metaPages: nil,
	}
}

func (l *waLog) makeTxWALState(limit uint) txWalState {
	if limit == 0 {
		// TODO: init wal limit on init, based on max file size
		limit = defaultWALLimit
	}

	return txWalState{
		walLimit: limit,
	}
}

func (l *waLog) Get(id PageID) PageID {
	return l.mapping[id]
}

func (l *waLog) fileCommitPrepare(st *walCommitState, tx *txWalState) {
	st.tx = tx
	newWal := createMappingUpdate(l.mapping, tx)
	st.checkpoint = tx.walLimit > 0 && uint(len(newWal)) >= tx.walLimit
	st.updated = st.checkpoint || tx.Updated()

	if st.checkpoint {
		newWal = tx.new
	}
	st.mapping = newWal
}

func (l *waLog) fileCommitAlloc(tx *Tx, st *walCommitState) error {
	if !st.updated {
		return nil
	}

	pages := predictWALMappingPages(st.mapping, uint(tx.PageSize()))
	if pages > 0 {
		st.allocRegions = tx.metaAllocator().AllocRegions(&tx.alloc, pages)
		if st.allocRegions == nil {
			return errOutOfMemory
		}
	}
	return nil
}

func (l *waLog) fileCommitSerialize(
	st *walCommitState,
	pageSize uint,
	onPage func(id PageID, buf []byte) error,
) error {
	if !st.updated {
		return nil
	}
	return writeWAL(st.allocRegions, pageSize, st.mapping, onPage)
}

func (l *waLog) fileCommitMeta(meta *metaPage, st *walCommitState) {
	if st.updated {
		var rootPage PageID
		if len(st.allocRegions) > 0 {
			rootPage = st.allocRegions[0].id
		}
		meta.wal.Set(rootPage)
	}
}

func (l *waLog) Commit(st *walCommitState) {
	if st.updated {
		l.mapping = st.mapping
		l.metaPages = st.allocRegions
	}
}

func (l walMapping) empty() bool {
	return len(l) == 0
}

func (s *txWalState) Release(id PageID) {
	s.free.Add(id)
	if s.new != nil {
		delete(s.new, id)
	}
}

func (s *txWalState) Updated() bool {
	return !s.free.Empty() || !s.new.empty()
}

func (s *txWalState) Set(orig, overwrite PageID) {
	if s.new == nil {
		s.new = walMapping{}
	}
	s.new[orig] = overwrite
}

func createMappingUpdate(old walMapping, tx *txWalState) walMapping {
	if !tx.Updated() {
		return nil
	}

	new := walMapping{}
	for id, walID := range old {
		if tx.free.Has(id) {
			continue
		}
		if _, exists := tx.new[id]; exists {
			continue
		}

		new[id] = walID
	}
	for id, walID := range tx.new {
		new[id] = walID
	}

	return new
}

func predictWALMappingPages(m walMapping, pageSize uint) uint {
	perPage := walEntriesPerPage(pageSize)
	return (uint(len(m)) + perPage - 1) / perPage
}

func walEntriesPerPage(pageSize uint) uint {
	payload := pageSize - walHeaderSize
	return payload / walEntrySize
}

func readWALMapping(
	wal *waLog,
	access func(PageID) []byte,
	root PageID,
) error {
	mapping, ids, err := readWAL(access, root)
	if err != nil {
		return nil
	}

	wal.mapping = mapping
	wal.metaPages = ids.Regions()
	return nil
}

func readWAL(
	access func(PageID) []byte,
	root PageID,
) (walMapping, idList, error) {
	if root == 0 {
		return walMapping{}, nil, nil
	}

	mapping := walMapping{}
	var metaPages idList
	for pageID := root; pageID != 0; {
		metaPages.Add(pageID)
		node, data := castWalPage(access(pageID))
		if node == nil {
			return nil, nil, errOutOfBounds
		}

		count := int(node.count.Get())
		pageID = node.next.Get()

		for i := 0; i < count; i++ {
			// read node mapping. Only 7 bytes are used per pageID
			var k, v pgID
			copy(k[0:7], data[0:7])
			copy(v[0:7], data[7:14])
			data = data[14:]

			mapping[k.Get()] = v.Get()
		}
	}

	return mapping, metaPages, nil
}

func writeWAL(
	to regionList,
	pageSize uint,
	mapping walMapping,
	onPage func(id PageID, buf []byte) error,
) error {
	allocPages := to.PageIDs()
	writer := newPagingWriter(allocPages, pageSize, 0, onPage)
	for id, walID := range mapping {
		var k, v pgID
		k.Set(id)
		v.Set(walID)

		var payload [walEntrySize]byte
		copy(payload[0:7], k[0:7])
		copy(payload[7:14], v[0:7])
		if err := writer.Write(payload[:]); err != nil {
			return err
		}
	}
	return writer.Flush()
}
