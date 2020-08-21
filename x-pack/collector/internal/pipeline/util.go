package pipeline

import (
	"context"
	"sync"

	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/unison"
)

type managedGroup struct {
	mu     sync.Mutex
	wg     unison.SafeWaitGroup
	active map[string]*processHandle
}

type processHandle struct {
	ctx    context.Context
	cancel func()
	wg     sync.WaitGroup
}

type cancelCtx struct {
	signaler unison.Canceler
	cancel   func()
}

func (grp *managedGroup) Go(name string, fn func(unison.Canceler)) {
	grp.mu.Lock()
	defer grp.mu.Unlock()

	if err := grp.wg.Add(1); err != nil {
		// already shutting down, do not spawn a go-routine
		return
	}

	if grp.active == nil {
		grp.active = map[string]*processHandle{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	hdl := &processHandle{ctx: ctx, cancel: cancel}
	hdl.wg.Add(1)
	grp.active[name] = hdl

	go func() {
		defer grp.wg.Done()
		defer hdl.wg.Done()
		defer func() {
			grp.mu.Lock()
			defer grp.mu.Unlock()
			delete(grp.active, name)
		}()
		defer cancel()

		fn(ctx)
	}()
}

func (grp *managedGroup) FindAll(pred func(string) bool) []*processHandle {
	grp.mu.Lock()
	defer grp.mu.Unlock()

	var handles []*processHandle
	for name, hdl := range grp.active {
		if pred(name) {
			handles = append(handles, hdl)
		}
	}
	return handles
}

func (grp *managedGroup) Has(name string) bool {
	grp.mu.Lock()
	defer grp.mu.Unlock()
	_, exists := grp.active[name]
	return exists
}

func (grp *managedGroup) Stop() {
	grp.signalStop()
	grp.wg.Wait()
}

func (grp *managedGroup) signalStop() {
	grp.mu.Lock()
	defer grp.mu.Unlock()

	grp.wg.Close()
	for _, hdl := range grp.active {
		hdl.cancel()
	}
	grp.active = nil
}

func cancelAll(handles []*processHandle) {
	for _, h := range handles {
		h.cancel()
	}
}

func waitAll(handles []*processHandle) {
	for _, h := range handles {
		h.wg.Wait()
	}
}

func backgroundCancelCtx() cancelCtx {
	return cancelCtx{
		signaler: context.Background(),
		cancel:   func() {},
	}
}

func makeCancelCtx(parent cancelCtx) cancelCtx {
	ctx, cancel := context.WithCancel(ctxtool.FromCanceller(parent.signaler))
	return cancelCtx{
		signaler: ctx,
		cancel:   cancel,
	}
}
