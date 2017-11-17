package socket

// hashSet is a simple set built upon a map.
type hashSet map[uint64]struct{}

// Add adds a value to the set.
func (s hashSet) Add(hash uint64) {
	s[hash] = struct{}{}
}

// Contains return true if the value is in the set.
func (s hashSet) Contains(hash uint64) bool {
	_, exists := s[hash]
	return exists
}

// Reset resets the contents of the set to empty and returns itself.
func (s hashSet) Reset() hashSet {
	for k := range s {
		delete(s, k)
	}
	return s
}
