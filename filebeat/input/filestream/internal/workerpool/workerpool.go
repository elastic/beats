package workerpool

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

type Pool struct {
	limit int

	work     chan func(context.Context) error
	workerWG *sync.WaitGroup
	goWG     *sync.WaitGroup
	errsCh   chan error

	errs     []error
	errsSize int
	errsNext int
	errsMu   *sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
}

type Goer interface {
	Go(fn func(context.Context) error) error
}

func New(limit int, maxErrors int) *Pool {
	if limit > 0 {
		return NewPool(limit, maxErrors)
	}

	panic("implement goroutine sea")
}

func NewPool(limit int, maxErrors int) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	work := make(chan func(context.Context) error)

	pool := Pool{
		limit:    limit,
		work:     work,
		workerWG: &sync.WaitGroup{},
		goWG:     &sync.WaitGroup{},
		errsSize: maxErrors,
		errs:     make([]error, maxErrors, maxErrors),
		errsCh:   make(chan error, limit),
		errsMu:   &sync.Mutex{},
		ctx:      ctx,
		cancel:   cancel,
	}

	// semaphore.NewWeighted(int64(limit))
	for i := 0; i < limit; i++ {
		pool.workerWG.Add(1)
		go func() {
			defer pool.workerWG.Done()

			for w := range work {
				err := w(ctx)
				if err != nil {
					pool.addErr(err)
				}
			}
		}()
	}

	return &pool
}

// Go starts fn on a goroutine when a worker becomes available.
// If the worker pool was already closed, Go returns a context.Canceled error.
// If there are no workers available and Pool.Stop() is called, fn is discarded.
// Go does not block.
func (p *Pool) Go(fn func(context.Context) error) error {
	if err := p.ctx.Err(); err != nil {
		return fmt.Errorf("worker pool is closed: %w", err)
	}

	// add the sempahore here

	p.goWG.Add(1)
	go func() {
		defer p.goWG.Done() // avoid sending on a closed channel
		select {
		case <-p.ctx.Done():
			p.addErr(fmt.Errorf(
				"could not send workload to be processed: worker pool is closed: %w",
				p.ctx.Err()))
		case p.work <- fn:
		}
	}()

	return nil
}

func (p *Pool) addErr(err error) {
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

func (p *Pool) Stop() error {
	p.cancel()

	// In order to avoid workload being sent on a closed channel, we need to
	// ensure no goroutine is waiting to send to p.work
	p.goWG.Wait()
	close(p.work)

	p.workerWG.Wait()
	p.errsMu.Lock()
	defer p.errsMu.Unlock()

	if p.errs[0] == nil {
		return nil
	}

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
