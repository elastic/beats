package txfile

import (
	"math"
	"sort"

	"github.com/elastic/go-txfile/internal/invariant"
	"github.com/elastic/go-txfile/internal/iter"
)

// freelist manages freed pages within an area. The freelist uses
// run-length-encoding, compining multiple pages into one region, so to reduce
// memory usage in memory, as well as when serializing the freelist.  The
// freelist guarantees pages are sorted by PageID. Depending on allocOrder,
// pages with smallest/biggest PageID will be allocated first.
type freelist struct {
	avail   uint
	regions regionList
}

// freelistEncPagePrediction is used to predict the number of meta-pages required
// to serialize the freelist.
// The prediction might over-estimate the number of pages required, which is
// perfectly fine, as long as we don't under-estimate (which would break
// serialization -> transaction fail).
type freelistEncPagePrediction struct {
	count              uint
	payloadSize, avail uint
}

// allocOrder provides freelist access strategies.
type allocOrder struct {
	// freelist iteration order
	iter iter.Fn

	// reportRange provides iteration limits for reporting allocated regions in
	// order (by PageID).
	reportRange func(last, len int) (int, int)

	// allocFromRegion split the region into allocated and leftover region.
	allocFromRegion func(reg region, N uint32) (region, region)

	// keepRange determines which pages to keep/remove from the freelist, after
	// allocation.
	keepRange func(last, len int, partial bool) (int, int)
}

var (
	allocFromBeginning = &allocOrder{
		iter: iter.Forward,
		reportRange: func(last, len int) (int, int) {
			return 0, last + 1
		},
		allocFromRegion: func(reg region, N uint32) (region, region) {
			return region{id: reg.id, count: N}, region{id: reg.id + PageID(N), count: reg.count - N}
		},
		keepRange: func(last, len int, partial bool) (int, int) {
			if partial {
				return last, len
			}
			return last + 1, len
		},
	}

	allocFromEnd = &allocOrder{
		iter: iter.Reversed,
		reportRange: func(last, len int) (int, int) {
			return last, len
		},
		allocFromRegion: func(reg region, N uint32) (region, region) {
			return region{id: reg.id + PageID(reg.count-N), count: N}, region{id: reg.id, count: reg.count - N}
		},
		keepRange: func(last, len int, partial bool) (int, int) {
			if partial {
				return 0, last + 1
			}
			return 0, last
		},
	}
)

// Avail returns number of pages available in the current freelist.
func (f *freelist) Avail() uint {
	return f.avail
}

// AllocAllRegionsWith allocates all regions in the freelist.
// The list will be empty afterwards.
func (f *freelist) AllocAllRegionsWith(fn func(region)) {
	for _, r := range f.AllocAllRegions() {
		fn(r)
	}
}

// AllocAllRegions allocates all regions in the freelist.
// The list will be empty afterwards.
func (f *freelist) AllocAllRegions() regionList {
	regions := f.regions
	f.avail = 0
	f.regions = nil
	return regions
}

// AllocContinuousRegion tries to find a contiuous set of pages in the freelist.
// The best-fitting (or smallest) region having at least n pages will be used
// for allocation.
// Returns an empty region, if no continuous space could be found.
func (f *freelist) AllocContinuousRegion(order *allocOrder, n uint) region {
	if f.avail < n || (f.avail == n && len(f.regions) > 1) {
		return region{}
	}

	if n > math.MaxUint32 { // continuous regions max out at 4GB
		return region{}
	}

	bestFit := -1
	bestSz := uint(math.MaxUint32)
	for i, end, next := order.iter(len(f.regions)); i != end; i = next(i) {
		count := uint(f.regions[i].count)
		if n <= count && count < bestSz {
			bestFit = i
			bestSz = count
			if bestSz == n {
				break
			}
		}
	}

	if bestFit < 0 {
		// no continuous region found
		return region{}
	}

	// allocate best fitting region from list
	i := bestFit
	selected := f.regions[i]
	allocated, rest := order.allocFromRegion(selected, uint32(n))

	invariant.Check(allocated.count == uint32(n), "allocation mismatch")
	invariant.Check(allocated.count+rest.count == selected.count, "region split page count mismatch")

	if rest.count == 0 {
		// remove entry
		copy(f.regions[i:], f.regions[i+1:])
		f.regions = f.regions[:len(f.regions)-1]
	} else {
		f.regions[i] = rest
	}

	f.avail -= uint(allocated.count)
	return allocated
}

// AllocRegionsWith allocates up n potentially non-continuous pages from the
// freelist. No page will be allocated, if n succeeds the number of available
// pages.
func (f *freelist) AllocRegionsWith(order *allocOrder, n uint, fn func(region)) {
	if n == 0 {
		return
	}

	var (
		last int // last region to allocate from
		L    = len(f.regions)
		N    = n // number of pages to be allocated from 'last' region
	)

	if N > f.avail {
		// not enough space -> return early
		return
	}

	// Collect indices of regions to be allocated from.
	for i, end, next := order.iter(L); i != end; i = next(i) {
		count := uint(f.regions[i].count)
		if count >= N {
			last = i
			break
		}
		N -= count
	}

	// Compute region split on last region to be allocated from.
	selected := f.regions[last]
	allocated, leftover := order.allocFromRegion(selected, uint32(N))

	invariant.Check(allocated.count == uint32(N), "allocation mismatch")
	invariant.Check(allocated.count+leftover.count == selected.count, "region split page count mismatch")

	// Implicitely update last allocated region to match the allocation size
	// and report all regions allocated.
	f.regions[last] = allocated
	for i, end := order.reportRange(last, L); i != end; i++ {
		fn(f.regions[i])
	}

	// update free regions
	f.regions[last] = leftover
	start, end := order.keepRange(last, L, leftover.count != 0)
	f.regions = f.regions[start:end]
	f.avail -= n
}

// AddRegions merges a new list of regions with the freelist. The regions
// in the list must be sorted.
func (f *freelist) AddRegions(list regionList) {
	count := list.CountPages()
	if count > 0 {
		f.regions = mergeRegionLists(f.regions, list)
		f.avail += count
	}
}

// AddRegion inserts a new region into the freelist. AddRegion ensures the new
// region is sorted within the freelist, potentially merging the new region
// with existing regions.
// Note: The region to be added MUST NOT overlap with existing regions.
func (f *freelist) AddRegion(reg region) {
	if len(f.regions) == 0 {
		f.regions = regionList{reg}
		f.avail += uint(reg.count)
		return
	}

	i := sort.Search(len(f.regions), func(i int) bool {
		_, end := f.regions[i].Range()
		return reg.id < end
	})

	total := uint(reg.count)
	switch {
	case len(f.regions) <= i: // add to end of region list?
		last := len(f.regions) - 1
		if regionsMergable(f.regions[last], reg) {
			f.regions[last] = mergeRegions(f.regions[last], reg)
		} else {
			f.regions.Add(reg)
		}
	case i == 0: // add to start of region list?
		if regionsMergable(reg, f.regions[0]) {
			f.regions[0] = mergeRegions(reg, f.regions[0])
		} else {
			f.regions = append(f.regions, region{})
			copy(f.regions[1:], f.regions)
			f.regions[0] = reg
		}
	default: // insert in middle of region list
		// try to merge region with already existing regions
		mergeBefore := regionsMergable(f.regions[i-1], reg)
		if mergeBefore {
			reg = mergeRegions(f.regions[i-1], reg)
		}
		mergeAfter := regionsMergable(reg, f.regions[i])
		if mergeAfter {
			reg = mergeRegions(reg, f.regions[i])
		}

		// update region list
		switch {
		case mergeBefore && mergeAfter: // combine adjacent regions -> shrink list
			f.regions[i-1] = reg
			copy(f.regions[i:], f.regions[i+1:])
			f.regions = f.regions[:len(f.regions)-1]
		case mergeBefore:
			f.regions[i-1] = reg
		case mergeAfter:
			f.regions[i] = reg
		default: // no adjacent entries -> grow list
			f.regions = append(f.regions, region{})
			copy(f.regions[i+1:], f.regions[i:])
			f.regions[i] = reg
		}
	}

	f.avail += total
}

// RemoveRegion removes all pages from the freelist, that are found within
// the input region.
func (f *freelist) RemoveRegion(removed region) {
	i := sort.Search(len(f.regions), func(i int) bool {
		_, end := f.regions[i].Range()
		return removed.id <= end
	})
	if i < 0 || i >= len(f.regions) {
		return
	}

	current := &f.regions[i]
	if current.id == removed.id && current.count == removed.count {
		// fast path: entry can be completely removed
		f.regions = append(f.regions[:i], f.regions[i+1:]...)
		f.avail -= uint(removed.count)
		return
	}

	var total uint
	removedStart, removedEnd := removed.Range()
	for removedStart < removedEnd && i < len(f.regions) {
		current := &f.regions[i]

		if removedStart < current.id {
			// Gap: advance removedStart, so to deal with holes when removing the all regions
			//      matching the input region
			removedStart = current.id
			continue
		}

		count := uint32(removedEnd - removedStart)
		if removedStart == current.id {
			if current.count < count {
				count = current.count
			}

			// remove entry:
			current.id = current.id + PageID(count)
			current.count -= count
			if current.count == 0 {
				// remove region from freelist -> i will point to next region if
				// `removed` overlaps 2 non-merged regions
				f.regions = append(f.regions[:i], f.regions[i+1:]...)
			} else {
				// overlapping region, but old region must be preserved:
				i++
			}

			removedStart += PageID(count)
			total += uint(count)
		} else {
			// split current region in removedStart
			keep := uint32(removedStart - current.id)
			leftover := region{
				id:    removedStart,
				count: current.count - keep,
			}
			current.count = keep

			// remove sub-region from leftover
			if leftover.count < count {
				count = leftover.count
			}
			leftover.id += PageID(count)
			leftover.count -= count

			total += uint(count)
			removedStart += PageID(count)
			i++ // advance to next region

			// insert new entry into regionList if removed did remove region in
			// middle of old region
			if leftover.count > 0 {
				f.regions = append(f.regions, region{})
				copy(f.regions[i+1:], f.regions[i:])
				f.regions[i] = leftover
				break // no more region to split from
			}
		}
	}

	f.avail -= total
}

// (de-)serialization

func readFreeList(
	access func(PageID) []byte,
	root PageID,
	fn func(bool, region),
) (idList, error) {
	if root == 0 {
		return nil, nil
	}

	rootPage := access(root)
	if rootPage == nil {
		return nil, errOutOfBounds
	}

	var metaPages idList
	for pageID := root; pageID != 0; {
		metaPages.Add(pageID)
		node, payload := castFreePage(access(pageID))
		if node == nil {
			return nil, errOutOfBounds
		}

		pageID = node.next.Get()
		entries := node.count.Get()
		tracef("free list node: (next: %v, entries: %v)", pageID, entries)

		for ; entries > 0; entries-- {
			isMeta, reg, n := decodeRegion(payload)
			payload = payload[n:]
			fn(isMeta, reg)
		}
	}

	return metaPages, nil
}

func writeFreeLists(
	to regionList,
	pageSize uint,
	metaList, dataList regionList,
	onPage func(id PageID, buf []byte) error,
) error {
	allocPages := to.PageIDs()
	writer := newPagingWriter(allocPages, pageSize, 0, onPage)

	var writeErr error
	writeList := func(isMeta bool, lst regionList) {
		if writeErr != nil {
			return
		}

		for _, reg := range lst {
			var buf [maxRegionEncSz]byte
			n := encodeRegion(buf[:], isMeta, reg)
			if err := writer.Write(buf[:n]); err != nil {
				writeErr = err
				return
			}
		}
	}

	writeList(true, metaList)
	writeList(false, dataList)
	if writeErr != nil {
		return writeErr
	}

	return writer.Flush()
}

func prepareFreelistEncPagePrediction(header int, pageSize uint) freelistEncPagePrediction {
	return freelistEncPagePrediction{payloadSize: pageSize - uint(header)}
}

func (f *freelistEncPagePrediction) Estimate() uint {
	return f.count
}

func (f *freelistEncPagePrediction) AddRegion(reg region) {
	sz := uint(regionEncodingSize(reg))
	if f.avail < sz {
		f.count++
		f.avail = f.payloadSize
	}
	f.avail -= sz
}

func (f *freelistEncPagePrediction) AddRegions(lst regionList) {
	for _, reg := range lst {
		f.AddRegion(reg)
	}
}
