package scheduler

type TimerHeap []*TimerTask

func (th TimerHeap) Less(i, j int) bool {
	return th[i].runAt.After(th[j].runAt)
}

func (th TimerHeap) Swap(i, j int) {
	th[i], th[j] = th[j], th[i]
}

func (th *TimerHeap) Push(tt interface{}) {
	*th = append(*th, tt.(*TimerTask))
}

func (th *TimerHeap) Pop() interface{} {
	old := *th
	n := len(old)
	tt := old[n-1]
	*th = old[0 : n-1]
	return tt
}

func (th TimerHeap) Len() int {
	return len(th)
}
