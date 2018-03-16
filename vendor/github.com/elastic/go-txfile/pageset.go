package txfile

type pageSet map[PageID]struct{}

func (s *pageSet) Add(id PageID) {
	if *s == nil {
		*s = pageSet{}
	}
	(*s)[id] = struct{}{}
}

func (s pageSet) Has(id PageID) bool {
	if s != nil {
		_, exists := s[id]
		return exists
	}
	return false
}

func (s pageSet) Empty() bool { return s.Count() == 0 }

func (s pageSet) Count() int { return len(s) }

func (s pageSet) IDs() idList {
	L := len(s)
	if L == 0 {
		return nil
	}

	l, i := make(idList, L), 0
	for id := range s {
		l[i], i = id, i+1
	}
	return l
}

func (s pageSet) Regions() regionList {
	if len(s) == 0 {
		return nil
	}

	regions, i := make(regionList, len(s)), 0
	for id := range s {
		regions[i], i = region{id: id, count: 1}, i+1
	}
	optimizeRegionList(&regions)

	return regions
}
