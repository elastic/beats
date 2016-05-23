package lj

type Batch struct {
	Events []interface{}
	ack    chan struct{}
}

func NewBatch(evts []interface{}) *Batch {
	return &Batch{evts, make(chan struct{})}
}

func (b *Batch) ACK() {
	close(b.ack)
}

func (b *Batch) Await() <-chan struct{} {
	return b.ack
}
