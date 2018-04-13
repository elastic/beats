package txfile

import (
	"math"

	"github.com/elastic/go-txfile/internal/invariant"
)

// file global allocator state
type (

	// allocator manages the on-disk page allocation. Pages in the allocator can
	// be either part of the meta-area or data-area. Users allocate pages from
	// the data-area only. The meta-area keeps pages available for in file
	// meta-data like overwrite pages and freelists. The meta-area allocates
	// pages from the data-area, if required. The meta-area grows by always doubling
	// the amount of pages in the meta-area.
	// For allocations one must get an instance to the dataAllocator,
	// walAllocator or metaAllocator respectively. Each allocator provides
	// slightly different allocation strategies.
	// The walAllocator is used for contents overwrite pages, while the
	// metaAllocator is used to allocate pages for for serializing the overwrite
	// mapping and freelispages for for serializing the overwrite mapping and
	// freelist.
	allocator struct {
		// configuration
		maxPages uint
		maxSize  uint
		pageSize uint

		// meta area
		meta      allocArea
		metaTotal uint // total number of pages reserved by meta area

		// data area
		data allocArea

		// allocator file metadata
		freelistRoot  PageID
		freelistPages regionList // page ids used to store the free list
	}

	allocArea struct {
		endMarker PageID
		freelist  freelist
	}

	// custom allocator implementations, sharing the global allocator state
	dataAllocator allocator // allocate from data area
	walAllocator  allocator // allocate WAL overwrite pages from beginning of meta area
	metaAllocator allocator // allocate meta pages from end of meta area

	// metaManager manages the data and meta regions, by moving regions
	// between those areas. The manager is used by walAllocator and metaAllocator
	// only.
	metaManager allocator
)

//transaction local allocation state
type (
	// txAllocState is used by write transactions, to record changes to the file
	// allocation state. The file global allocator state is modified within the
	// write transaction. txAllocState acts as undo/redo-log for the in-memory
	// allocation state.
	// Page frees are only recorded within the transaction. No pages are returned
	// to the allocator, so to ensure a page freed can not be allocated. This
	// guarantees freed pages can not be overwritten in the current transaction
	// (keep most recent transaction intact).
	txAllocState struct {
		manager txAreaManageState
		data    txAllocArea
		meta    txAllocArea
		options txAllocOptions // per transaction allocation options
	}

	txAllocArea struct {
		endMarker PageID
		allocated pageSet // allocated pages from freelist
		new       pageSet // allocated pages from end of file
		freed     pageSet // set of pages freed within transaction
	}

	txAreaManageState struct {
		moveToMeta regionList // list regions moved from data area to meta area
	}

	// txAllocOptions keeps track of user options passed to the transaction.
	txAllocOptions struct {
		overflowAreaEnabled bool // enable allocating pages with ID > maxPages for metadata
		metaGrowPercentage  int  // limit of meta area in use, so to allocate new pages into the meta area
	}
)

// allocCommitState keeps track of the new allocator state during the commit.
// These changes must be recorded for now, as the final allocator state must
// not be updated in memory until after the transaction has been commited to
// the file.
type allocCommitState struct {
	tx            *txAllocState
	updated       bool       // set if updates to allocator within current transaction
	allocRegions  regionList // meta pages allocated to write new freelist too
	metaList      regionList // new meta area freelist
	dataList      regionList // new data area freelist
	overflowFreed uint       // number of pages in overflow region to be returned
}

// noLimit indicates the data/meta-area can grow without any limits.
const noLimit uint = maxUint

const defaultMetaGrowPercentage = 80

// allocator
// ---------

func (a *allocator) DataAllocator() *dataAllocator   { return (*dataAllocator)(a) }
func (a *allocator) WALPageAllocator() *walAllocator { return (*walAllocator)(a) }
func (a *allocator) MetaAllocator() *metaAllocator   { return (*metaAllocator)(a) }
func (a *allocator) metaManager() *metaManager       { return (*metaManager)(a) }

func (a *allocator) makeTxAllocState(withOverflow bool, growPercentage int) txAllocState {
	if growPercentage <= 0 {
		growPercentage = defaultMetaGrowPercentage
	}

	return txAllocState{
		data: txAllocArea{
			endMarker: a.data.endMarker,
		},
		meta: txAllocArea{
			endMarker: a.meta.endMarker,
		},
		options: txAllocOptions{
			overflowAreaEnabled: withOverflow,
			metaGrowPercentage:  growPercentage,
		},
	}
}

func (a *allocator) fileCommitPrepare(st *allocCommitState, tx *txAllocState) {
	st.tx = tx
	st.updated = tx.Updated()
	if st.updated {
		a.MetaAllocator().FreeRegions(tx, a.freelistPages)
	}
}

func (a *allocator) fileCommitAlloc(st *allocCommitState) error {
	if !st.updated {
		return nil
	}

	dataFreed := st.tx.data.freed.Regions()
	metaFreed := st.tx.meta.freed.Regions()

	// Predict number of meta pages required to store new freelist,
	// by iterating all region entries and take the potential encoding size
	// into account. As allocation might force a region from the data area
	// being moved (or split) into the meta area, we add more dummy region
	// with enforced max size. So the allocator can move pages between
	// meta and data if required.
	// This method over-estimates the number of required pages, as
	// we will have to allocate pages from the metaFree lists end
	// after the estimator finishes.
	prediction := prepareFreelistEncPagePrediction(freePageHeaderSize, a.pageSize)
	prediction.AddRegions(dataFreed)
	prediction.AddRegions(metaFreed)
	prediction.AddRegions(a.data.freelist.regions)
	prediction.AddRegions(a.meta.freelist.regions)
	if prediction.count > 0 {
		// only add extra pages if we need to write the meta page
		prediction.AddRegion(region{id: 1, count: math.MaxUint32})
		prediction.AddRegion(region{id: 1, count: math.MaxUint32})
	}

	// alloc regions for writing the new freelist
	var allocRegions regionList
	if n := prediction.count; n > 0 {
		allocRegions = a.MetaAllocator().AllocRegions(st.tx, n)
		if allocRegions == nil {
			return errOutOfMemory
		}
	}

	// Compute new freelist. As consecutive regions are merged the
	// resulting list might require less pages
	newDataList := mergeRegionLists(a.data.freelist.regions, dataFreed)
	newMetaList := mergeRegionLists(a.meta.freelist.regions, metaFreed)

	st.allocRegions = allocRegions
	st.dataList = newDataList

	// remove pages from end of overflow area from meta freelist + adjust end marker
	st.metaList, st.overflowFreed = releaseOverflowPages(newMetaList, a.maxPages, a.meta.endMarker)
	return nil
}

// releaseOverflowPages removes pages at the end of a region list as long as
// the current end marker is bigger then the maximum number of allowed pages
// and the freelist contains some continuous regions up to endMarker.
func releaseOverflowPages(
	list regionList,
	maxPages uint, endMarker PageID,
) (regionList, uint) {
	overflowStart, overflowEnd := PageID(maxPages), endMarker
	if maxPages == 0 || overflowStart >= overflowEnd {
		return list, 0
	}

	var freed uint
	for i := len(list) - 1; i != -1; i-- {
		start, end := list[i].Range()
		if end < overflowEnd {
			break
		}

		if start < overflowStart {
			// split
			list[i].count = uint32(overflowStart - start)
			freed += uint(end - overflowStart)
			overflowEnd = overflowStart
		} else {
			// remove range
			overflowEnd = start
			freed += uint(list[i].count)
			list = list[:i]
		}
	}

	return list, freed
}

func (a *allocator) fileCommitSerialize(
	st *allocCommitState,
	onPage func(id PageID, buf []byte) error,
) error {
	if !st.updated || len(st.allocRegions) == 0 {
		return nil
	}
	return writeFreeLists(st.allocRegions, a.pageSize, st.metaList, st.dataList, onPage)
}

func (a *allocator) fileCommitMeta(meta *metaPage, st *allocCommitState) {
	if st.updated {
		var freelistRoot PageID
		if len(st.allocRegions) > 0 {
			freelistRoot = st.allocRegions[0].id
		}
		meta.freelist.Set(freelistRoot)

		dataEndMarker := a.data.endMarker
		metaEndMarker := a.meta.endMarker
		if st.overflowFreed > 0 {
			metaEndMarker -= PageID(st.overflowFreed)
			if metaEndMarker > dataEndMarker {
				dataEndMarker = metaEndMarker
			}
		}

		meta.dataEndMarker.Set(dataEndMarker)
		meta.metaEndMarker.Set(metaEndMarker)
		meta.metaTotal.Set(uint64(a.metaTotal - st.overflowFreed))
	}
}

func (a *allocator) Commit(st *allocCommitState) {
	if st.updated {
		a.freelistPages = st.allocRegions
		if len(st.allocRegions) > 0 {
			a.freelistRoot = st.allocRegions[0].id
		} else {
			a.freelistRoot = 0
		}

		a.data.commit(st.dataList)
		a.meta.commit(st.metaList)
		a.metaTotal -= st.overflowFreed
	}
}

func (a *allocator) Rollback(st *txAllocState) {
	// restore meta area
	a.meta.rollback(&st.meta)
	for _, reg := range st.manager.moveToMeta {
		a.meta.freelist.RemoveRegion(reg)
		a.metaTotal -= uint(reg.count)

		if reg.id < st.data.endMarker {
			reg.EachPage(st.data.allocated.Add)
		}
	}

	// restore data area
	a.data.rollback(&st.data)
}

func (a *allocArea) commit(regions regionList) {
	a.freelist.regions = regions
	a.freelist.avail = regions.CountPages()
}

func (a *allocArea) rollback(st *txAllocArea) {
	for id := range st.allocated {
		if id >= st.endMarker {
			delete(st.allocated, id)
		}
	}
	a.freelist.AddRegions(st.allocated.Regions())
	a.endMarker = st.endMarker
}

// metaManager
// -----------

func (mm *metaManager) DataAllocator() *dataAllocator {
	return (*dataAllocator)(mm)
}

func (mm *metaManager) Avail(st *txAllocState) uint {
	dataAvail := mm.DataAllocator().Avail(st)
	if dataAvail == noLimit || st.options.overflowAreaEnabled {
		return noLimit
	}

	return mm.meta.freelist.Avail() + dataAvail
}

func (mm *metaManager) Ensure(st *txAllocState, n uint) bool {
	total := mm.metaTotal
	avail := mm.meta.freelist.Avail()
	used := total - avail
	targetUsed := used + n

	invariant.Check(total >= avail, "invalid meta total page count")

	tracef("ensure(%v): total=%v, avail=%v, used=%v, targetUsed=%v\n",
		n, total, avail, used, targetUsed)

	pctGrow := st.options.metaGrowPercentage
	pctShrink := pctGrow / 2

	szMinMeta, szMaxMeta := metaAreaTargetQuota(total, targetUsed, pctShrink, pctGrow)
	traceln("  target quota: ", szMinMeta, szMaxMeta)

	invariant.Check(szMaxMeta >= szMinMeta, "required page count must grow")

	if szMaxMeta == total {
		// we still have enough memory in the meta area -> return success

		// TODO: allow 'ensure' to shrink the meta area
		return true
	}

	invariant.Check(szMaxMeta > total, "expected new page count exceeding allocated pages")

	// try to move regions from data area into the meta area:
	requiredMax := szMaxMeta - total
	if mm.tryGrow(st, requiredMax, false) {
		return true
	}

	// Can not grow until 'requiredMax' -> try to grow up to requiredMin,
	// potentially allocating pages from the overflow area
	requiredMin := szMinMeta - total
	if mm.tryGrow(st, requiredMin, st.options.overflowAreaEnabled) {
		return true
	}

	// out of memory
	return false
}

func (mm *metaManager) tryGrow(
	st *txAllocState,
	count uint,
	withOverflow bool,
) bool {
	da := mm.DataAllocator()
	avail := da.Avail(st)

	tracef("try grow meta area pages=%v, avail=%v\n", count, avail)

	if count == 0 {
		return true
	}

	if avail < count {
		if !withOverflow {
			traceln("can not grow meta area yet")
			return false
		}

		da.AllocRegionsWith(st, avail, func(reg region) {
			st.manager.moveToMeta.Add(reg)
			mm.metaTotal += uint(reg.count)
			mm.meta.freelist.AddRegion(reg)
		})

		// allocate from overflow area
		required := count - avail
		if required > 0 {
			traceln("try to grow overflow area")
		}
		allocFromArea(&st.meta, &mm.meta.endMarker, required, func(reg region) {
			// st.manager.fromOverflow.Add(reg)
			mm.metaTotal += uint(reg.count)
			mm.meta.freelist.AddRegion(reg)
		})
		if mm.maxPages == 0 && mm.data.endMarker < mm.meta.endMarker {
			mm.data.endMarker = mm.meta.endMarker
		}

		return true
	}

	// Enough memory available in data area. Try to allocate continuous region first
	reg := da.AllocContinuousRegion(st, count)
	if reg.id != 0 {
		st.manager.moveToMeta.Add(reg)
		mm.metaTotal += uint(reg.count)
		mm.meta.freelist.AddRegion(reg)
		return true
	}

	// no continuous memory block -> allocate single regions
	n := da.AllocRegionsWith(st, count, func(reg region) {
		st.manager.moveToMeta.Add(reg)
		mm.metaTotal += uint(reg.count)
		mm.meta.freelist.AddRegion(reg)
	})
	return n == count
}

func (mm *metaManager) Free(st *txAllocState, id PageID) {
	// mark page as freed for now
	st.meta.freed.Add(id)
}

func metaAreaTargetQuota(
	total, used uint,
	shrinkPercentage, growPercentage int,
) (min, max uint) {
	min = used
	max = uint(nextPowerOf2(uint64(used)))
	if max < total {
		max = total
	}

	usage := 100 * float64(used) / float64(max)

	// grow 'max' by next power of 2, if used area would exceed growPercentage
	needsGrow := usage > float64(growPercentage)

	// If memory is to be freed (max < total), still grow 'max' by next power of
	// 2 (so not to free too much memory at once), if used area in new meta area
	// would exceed shrinkPercentage.
	// => percentage of used area in new meta area will be shrinkPercentage/2
	needsGrow = needsGrow || (max < total && usage > float64(shrinkPercentage))

	if min < total {
		min = total
	}

	if needsGrow {
		max = max * 2
	}
	return min, max
}

// dataAllocator
// -------------

func (a *dataAllocator) Avail(_ *txAllocState) uint {
	if a.maxPages == 0 {
		return noLimit
	}
	return a.maxPages - uint(a.data.endMarker) + a.data.freelist.Avail()
}

func (a *dataAllocator) AllocContinuousRegion(
	st *txAllocState,
	n uint,
) region {
	avail := a.Avail(st)
	if avail < n {
		return region{}
	}

	reg := allocContFromFreelist(&a.data.freelist, &st.data, allocFromBeginning, n)
	if reg.id != 0 {
		return reg
	}

	avail = a.maxPages - uint(a.data.endMarker)
	if avail < n {
		// out of memory
		return region{}
	}

	allocFromArea(&st.data, &a.data.endMarker, n, func(r region) { reg = r })
	if a.meta.endMarker < a.data.endMarker {
		a.meta.endMarker = a.data.endMarker
	}
	return reg
}

func (a *dataAllocator) AllocRegionsWith(
	st *txAllocState,
	n uint,
	fn func(region),
) uint {
	avail := a.Avail(st)
	if avail < n {
		return 0
	}

	// Enough space available -> allocate all pages.
	count := n

	// 1. allocate subset of regions from freelist
	n -= allocFromFreelist(&a.data.freelist, &st.data, allocFromBeginning, n, fn)
	if n > 0 {
		// 2. allocate from yet unused data area
		allocFromArea(&st.data, &a.data.endMarker, n, fn)
		if a.meta.endMarker < a.data.endMarker {
			a.meta.endMarker = a.data.endMarker
		}
	}
	return count
}

func (a *dataAllocator) Free(st *txAllocState, id PageID) {
	traceln("free page:", id)

	if id < 2 || id >= a.data.endMarker {
		panic(errOutOfBounds)
	}

	if !st.data.new.Has(id) {
		// fast-path, page has not been allocated in current transaction
		st.data.freed.Add(id)
		return
	}

	// page has been allocated in current transaction -> return to allocator for immediate re-use
	a.data.freelist.AddRegion(region{id: id, count: 1})

	if st.data.endMarker >= id {
		// allocation from within old data region
		return
	}

	// allocation was from past the old end-marker. Check if we can shrink the
	// end marker again
	regions := a.data.freelist.regions
	last := len(regions) - 1
	start, end := regions[last].Range()
	if end < a.data.endMarker {
		// in middle of new data region -> can not adjust end marker -> keep update to freelist
		return
	}

	if st.data.endMarker > start {
		start = st.data.endMarker
		count := uint(end - start)
		regions[last].count -= uint32(count)
		a.data.freelist.avail -= count
	} else {
		a.data.freelist.avail -= uint(regions[last].count)
		a.data.freelist.regions = regions[:last]
	}
	a.data.endMarker = start
}

// walAllocator
// ------------

func (a *walAllocator) metaManager() *metaManager { return (*metaManager)(a) }

func (a *walAllocator) Avail(st *txAllocState) uint {
	return a.metaManager().Avail(st)
}

func (a *walAllocator) Alloc(st *txAllocState) PageID {
	mm := a.metaManager()
	if !mm.Ensure(st, 1) {
		return 0
	}

	// Use AllocContinuousRegion to find smallest fitting region
	// to allocate from.
	reg := a.meta.freelist.AllocContinuousRegion(allocFromBeginning, 1)
	if reg.id == 0 {
		return 0
	}
	st.meta.allocated.Add(reg.id)
	return reg.id
}

func (a *walAllocator) AllocRegionsWith(st *txAllocState, n uint, fn func(region)) uint {
	mm := a.metaManager()
	if !mm.Ensure(st, n) {
		return 0
	}

	return allocFromFreelist(&a.meta.freelist, &st.meta, allocFromBeginning, n, fn)
}

func (a *walAllocator) Free(st *txAllocState, id PageID) {
	a.metaManager().Free(st, id)
}

// metaAllocator
// ------------

func (a *metaAllocator) metaManager() *metaManager { return (*metaManager)(a) }

func (a *metaAllocator) Avail(st *txAllocState) uint {
	return a.metaManager().Avail(st)
}

func (a *metaAllocator) AllocRegionsWith(
	st *txAllocState,
	n uint,
	fn func(region),
) uint {
	mm := a.metaManager()
	if !mm.Ensure(st, n) {
		return 0
	}

	return allocFromFreelist(&a.meta.freelist, &st.meta, allocFromEnd, n, fn)
}

func (a *metaAllocator) AllocRegions(st *txAllocState, n uint) regionList {
	reg := make(regionList, 0, n)
	if n := a.AllocRegionsWith(st, n, reg.Add); n == 0 {
		return nil
	}
	return reg
}

func (a *metaAllocator) Free(st *txAllocState, id PageID) {
	a.metaManager().Free(st, id)
}

func (a *metaAllocator) FreeAll(st *txAllocState, ids idList) {
	for _, id := range ids {
		a.Free(st, id)
	}
}

func (a *metaAllocator) FreeRegions(st *txAllocState, regions regionList) {
	regions.EachPage(func(id PageID) {
		a.Free(st, id)
	})
}

// tx allocation state methods
// ---------------------------

func (s *txAllocState) Updated() bool {
	return s.meta.Updated() || s.data.Updated()
}

func (s *txAllocArea) Updated() bool {
	return !s.allocated.Empty() || !s.new.Empty() || !s.freed.Empty()
}

// allocator state (de-)serialization
// ----------------------------------

func readAllocatorState(a *allocator, f *File, meta *metaPage, opts Options) error {
	if a.maxSize > 0 {
		a.maxPages = a.maxSize / a.pageSize
	}

	a.data.endMarker = meta.dataEndMarker.Get()
	a.meta.endMarker = meta.metaEndMarker.Get()
	a.metaTotal = uint(meta.metaTotal.Get())

	a.freelistRoot = meta.freelist.Get()
	if a.freelistRoot == 0 {
		return nil
	}

	var metaList, dataList freelist
	ids, err := readFreeList(f.mmapedPage, a.freelistRoot, func(isMeta bool, region region) {
		lst := &dataList
		if isMeta {
			lst = &metaList
		}

		lst.avail += uint(region.count)
		lst.regions.Add(region)
	})
	if err != nil {
		return err
	}

	dataList.regions.Sort()
	dataList.regions.MergeAdjacent()
	metaList.regions.Sort()
	metaList.regions.MergeAdjacent()

	a.data.freelist = dataList
	a.meta.freelist = metaList
	a.freelistPages = ids.Regions()
	return nil
}

// allocator helpers/utilities
// ---------------------------

// allocFromFreelist allocates up to 'max' pages from the free list.
// The number of allocated pages is returned
func allocFromFreelist(
	f *freelist,
	area *txAllocArea,
	order *allocOrder,
	max uint,
	fn func(region),
) uint {
	count := max
	if f.avail < count {
		count = f.avail
	}

	f.AllocRegionsWith(order, count, func(region region) {
		region.EachPage(area.allocated.Add)
		fn(region)
	})
	return count
}

func allocContFromFreelist(
	f *freelist,
	area *txAllocArea,
	order *allocOrder,
	n uint,
) region {
	region := f.AllocContinuousRegion(order, n)
	if region.id != 0 {
		region.EachPage(area.new.Add)
	}
	return region
}

func allocFromArea(area *txAllocArea, marker *PageID, count uint, fn func(region)) {
	// region can be max 2<<32 -> allocate in loop
	id := *marker
	for count > 0 {
		n := count
		if n > math.MaxUint32 {
			n = math.MaxUint32
		}

		region := region{id: id, count: uint32(n)}
		region.EachPage(area.new.Add)
		fn(region)

		id += PageID(n)
		count -= n
	}
	*marker = id
}
