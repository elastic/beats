package consumergroup

type nameSet map[string]struct{}

func makeNameSet(strings ...string) nameSet {
	if len(strings) == 0 {
		return nil
	}

	set := nameSet{}
	for _, s := range strings {
		set[s] = struct{}{}
	}
	return set
}

func (s nameSet) has(name string) bool {
	if s == nil {
		return true
	}

	_, ok := s[name]
	return ok
}

func (s nameSet) pred() func(string) bool {
	if s == nil || len(s) == 0 {
		return nil
	}
	return s.has
}
