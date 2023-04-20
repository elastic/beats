package task

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/sync/semaphore"
)

type Group struct {
	limit uint64

	sem *semaphore.Weighted
	wg  *sync.WaitGroup

	errs     []error
	errsSize uint64
	errsNext uint64
	errsMu   *sync.Mutex

	ctx       context.Context
	cancelCtx context.CancelFunc
}

type Goer interface {
	Go(fn func(context.Context) error) error
}

func NewGroup(limit uint64, maxErrors uint64) *Group {
	if maxErrors < 1 {
		maxErrors = 1
	}
	ctx, cancel := context.WithCancel(context.Background())

	g := Group{
		limit:     limit,
		wg:        &sync.WaitGroup{},
		errsSize:  maxErrors,
		errs:      make([]error, maxErrors),
		errsMu:    &sync.Mutex{},
		ctx:       ctx,
		cancelCtx: cancel,
	}

	if limit > 0 {
		g.sem = semaphore.NewWeighted(int64(limit))
	}

	return &g
}

// Go starts fn on a goroutine when a worker becomes available.
// If the worker pool was already closed, Go returns a context.Canceled error.
// If there are no workers available and Group.Stop() is called, fn is discarded.
// Go does not block.
func (p *Group) Go(fn func(context.Context) error) error {
	if err := p.ctx.Err(); err != nil {
		return fmt.Errorf("task group is closed: %w", err)
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		if p.limit != 0 {
			defer p.sem.Release(1)
			err := p.sem.Acquire(p.ctx, 1)
			if err != nil {
				p.addErr(err)
				return
			}
		}

		p.addErr(fn(p.ctx))
	}()

	return nil
}

func (p *Group) addErr(err error) {
	if err == nil {
		return
	}

	p.errsMu.Lock()
	defer p.errsMu.Unlock()

	p.errs[p.errsNext] = err
	p.errsNext++

	if p.errsNext >= p.errsSize {
		p.errsNext = 0
	}
}

func (p *Group) Stop() error {
	p.cancelCtx()

	p.wg.Wait()
	p.errsMu.Lock()
	defer p.errsMu.Unlock()

	if p.errs[0] == nil {
		return nil
	}

	// TODO: use errors.Join once we update to go1.20
	buf := &strings.Builder{}
	buf.WriteString("last errors:")
	for i, err := range p.errs {
		if err == nil {
			break
		}

		if i == 0 {
			// safe to ignore as strings.Builder.Write does not fail.
			_, _ = fmt.Fprintf(buf, " %v", err)
			continue
		}

		// safe to ignore as strings.Builder.Write does not fail.
		_, _ = fmt.Fprintf(buf, " | %v", err)
	}

	return errors.New(buf.String())
}
