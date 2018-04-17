package txfile

import (
	"sort"
)

// region values represent a continuous set of pages.
type region struct {
	id    PageID
	count uint32
}

type regionList []region

// PageIDs represent pages. The minimal page size is 512Bytes +
// all contents in a file must be addressable by offset. This gives us
// 9 bytes to store additional flags or value in the entry.

const (
	maxRegionEncSz      = 8 + 4
	entryBits           = 9            // region entry header size in bits
	entryOverflow       = (1 << 8) - 1 // overflow marker == all counter bits set
	entryOverflowMarker = uint64(entryOverflow) << (64 - entryBits)
	entryCounterMask    = uint32(((1 << (entryBits - 1)) - 1))
	entryMetaFlag       = 1 << 63 // indicates the region holding pages used by the meta-area
)

func (l regionList) Len() int {
	return len(l)
}

func (l *regionList) Add(reg region) {
	*l = append(*l, reg)
}

func (l regionList) Sort() {
	sort.Slice(l, func(i, j int) bool {
		return l[i].Before(l[j])
	})
}

func (l *regionList) MergeAdjacent() {
	if len(*l) <= 1 {
		return
	}

	tmp := (*l)[:1]
	i := 0
	for _, r := range (*l)[1:] {
		if regionsMergable(tmp[i], r) {
			tmp[i] = mergeRegions(tmp[i], r)
		} else {
			tmp = append(tmp, r)
			i = i + 1
		}
	}
	*l = tmp
}

func (l regionList) CountPagesUpTo(id PageID) (count uint) {
	for _, reg := range l {
		if reg.id >= id {
			break
		}

		start, end := reg.Range()
		if end > id {
			end = id
		}
		count += uint(end - start)
	}
	return
}

func (l regionList) CountPages() (count uint) {
	for _, reg := range l {
		count += uint(reg.count)
	}
	return
}

func (l regionList) EachPage(fn func(PageID)) {
	for _, reg := range l {
		reg.EachPage(fn)
	}
}

func (l regionList) EachRegion(fn func(region)) {
	for _, reg := range l {
		fn(reg)
	}
}

func (l regionList) PageIDs() (ids idList) {
	l.EachPage(ids.Add)
	return
}

func (r region) Start() PageID           { return r.id }
func (r region) End() PageID             { return r.id + PageID(r.count) }
func (r region) Range() (PageID, PageID) { return r.Start(), r.End() }
func (r region) InRange(id PageID) bool  { return r.Start() <= id && id < r.End() }

func (r region) SplitAt(id PageID) region {
	start, end := r.Range()
	if id <= start || end < id {
		return region{}
	}

	if end > id {
		end = id
	}

	return region{id: start, count: uint32(end - start)}
}

func (r region) EachPage(fn func(PageID)) {
	for id, end := r.Range(); id != end; id++ {
		fn(id)
	}
}

func (r region) Before(other region) bool {
	return r.id < other.id
}

func (r region) Precedes(other region) bool {
	return r.id+PageID(r.count) == other.id
}

func mergeRegions(a, b region) region {
	return region{id: a.id, count: a.count + b.count}
}

// mergeRegionLists merges 2 sorter regionLists into a new sorted region list.
// Adjacent regions will be merged into a single region as well.
func mergeRegionLists(a, b regionList) regionList {
	L := len(a) + len(b)
	if L == 0 {
		return nil
	}

	final := make(regionList, 0, L)
	for len(a) > 0 && len(b) > 0 {
		if a[0].Before(b[0]) {
			final, a = append(final, a[0]), a[1:]
		} else {
			final, b = append(final, b[0]), b[1:]
		}
	}

	// copy leftover elements
	final = append(final, a...)
	final = append(final, b...)

	final.MergeAdjacent()

	return final
}

// regionsMergable checks region a directly precedes regions b and
// the region counter will not overflow.
func regionsMergable(a, b region) bool {
	if !a.Before(b) {
		a, b = b, a
	}
	return a.Precedes(b) && (a.count+b.count) > a.count
}

// optimizeRegionList sorts and merges adjacent regions.
func optimizeRegionList(reg *regionList) {
	initLen := reg.Len()
	reg.Sort()
	reg.MergeAdjacent()
	if l := reg.Len(); initLen > l {
		tmp := make(regionList, l, l)
		copy(tmp, *reg)
		*reg = tmp
	}
}

// (de-)serialization
// ------------------

func regionEncodingSize(r region) int {
	if r.count < entryOverflow {
		return (&u64{}).Len()
	}
	return (&u64{}).Len() + (&u32{}).Len()
}

func encodeRegion(buf []byte, isMeta bool, reg region) int {
	flag := uint64(0)
	if isMeta {
		flag = entryMetaFlag
	}

	payload := buf
	entry := castU64(payload)
	payload = payload[entry.Len():]

	if reg.count < entryOverflow {
		count := uint64(reg.count) << (64 - entryBits)
		entry.Set(flag | count | uint64(reg.id))
	} else {
		count := castU32(payload)
		payload = payload[count.Len():]

		entry.Set(flag | entryOverflowMarker | uint64(reg.id))
		count.Set(reg.count)
	}

	return len(buf) - len(payload)
}

func decodeRegion(buf []byte) (bool, region, int) {
	payload := buf
	entry := castU64(payload)
	value := entry.Get()
	payload = payload[entry.Len():]

	id := PageID((value << entryBits) >> entryBits)
	isMeta := (entryMetaFlag & value) == entryMetaFlag
	count := uint32(value>>(64-entryBits)) & entryCounterMask
	switch count {
	case 0:
		count = 1
	case entryOverflow:
		extra := castU32(payload)
		count, payload = extra.Get(), payload[extra.Len():]
	}

	return isMeta, region{id: id, count: count}, len(buf) - len(payload)
}
