package common

import (
	"sync"
)

// WorkerSignal ensure all events have been
// treated before closing Go routines
type WorkerSignal struct {
	Done     chan struct{}
	wgEvent  sync.WaitGroup
	wgWorker sync.WaitGroup
}

func NewWorkerSignal() *WorkerSignal {
	w := &WorkerSignal{}
	w.Init()
	return w
}

func (ws *WorkerSignal) Init() {
	ws.Done = make(chan struct{})
}

func (ws *WorkerSignal) AddEvent(delta int) {
	ws.wgEvent.Add(delta)
}

func (ws *WorkerSignal) DoneEvent() {
	ws.wgEvent.Done()
}

func (ws *WorkerSignal) WorkerStart() {
	ws.wgWorker.Add(1)
}

func (ws *WorkerSignal) WorkerFinished() {
	ws.wgWorker.Done()
}

func (ws *WorkerSignal) Stop() {
	ws.wgEvent.Wait()  // Wait for all events to be dealt with
	close(ws.Done)     // Ask Go routines to exit
	ws.wgWorker.Wait() // Wait for Go routines to finish
}
