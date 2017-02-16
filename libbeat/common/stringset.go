package common

type StringSet map[string]struct{}

func MakeStringSet(strings ...string) StringSet {
	if len(strings) == 0 {
		return nil
	}

	set := StringSet{}
	for _, str := range strings {
		set[str] = struct{}{}
	}
	return set
}

func (set StringSet) Add(s string) {
	set[s] = struct{}{}
}

func (set StringSet) Del(s string) {
	delete(set, s)
}

func (set StringSet) Count() int {
	return len(set)
}

func (set StringSet) Has(s string) (exists bool) {
	if set != nil {
		_, exists = set[s]
	}
	return
}
