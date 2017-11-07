package scheduler

import "sort"

type timeOrd []*job

func sortEntries(es []*job) {
	sort.Sort(timeOrd(es))
}

func (b timeOrd) Len() int {
	return len(b)
}

func (b timeOrd) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// Less reports `earliest` time i should sort before j.
// zero time is not `earliest` time.
func (b timeOrd) Less(i, j int) bool {
	if b[i].next.IsZero() {
		return false
	}
	if b[j].next.IsZero() {
		return true
	}
	return b[i].next.Before(b[j].next)
}
