package txfile

import "sort"

type idList []PageID

func (l *idList) Add(id PageID) {
	*l = append(*l, id)
}

func (l idList) ToSet() pageSet {
	L := len(l)
	if L == 0 {
		return nil
	}

	s := make(pageSet, L)
	for _, id := range l {
		s.Add(id)
	}
	return s
}

func (l idList) Sort() {
	sort.Slice(l, func(i, j int) bool {
		return l[i] < l[j]
	})
}

func (l idList) Regions() regionList {
	if len(l) == 0 {
		return nil
	}

	regions := make(regionList, len(l))
	for i, id := range l {
		regions[i] = region{id: id, count: 1}
	}
	optimizeRegionList(&regions)
	return regions
}
