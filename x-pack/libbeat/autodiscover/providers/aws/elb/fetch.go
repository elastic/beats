// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elb

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/elasticloadbalancingv2iface"
	"go.uber.org/multierr"

	"github.com/menderesk/beats/v7/libbeat/logp"
)

// fetcher is an interface that can fetch a list of lbListener (load balancer + listener) objects without pagination being necessary.
type fetcher interface {
	fetch(ctx context.Context) ([]*lbListener, error)
}

// apiMultiFetcher fetches results from multiple clients concatenating their results together
// Useful since we have a fetcher per region, this combines them.
type apiMultiFetcher struct {
	fetchers []fetcher
}

func (amf *apiMultiFetcher) fetch(ctx context.Context) ([]*lbListener, error) {
	fetchResults := make(chan []*lbListener)
	fetchErr := make(chan error)

	// Simultaneously fetch all from each region
	for _, f := range amf.fetchers {
		go func(f fetcher) {
			fres, ferr := f.fetch(ctx)
			if ferr != nil {
				fetchErr <- ferr
			} else {
				fetchResults <- fres
			}
		}(f)
	}

	var results []*lbListener
	var errs []error

	for pending := len(amf.fetchers); pending > 0; pending-- {
		select {
		case r := <-fetchResults:
			results = append(results, r...)
		case e := <-fetchErr:
			errs = append(errs, e)
		}
	}

	return results, multierr.Combine(errs...)
}

// apiFetcher is a concrete implementation of fetcher that hits the real AWS API.
type apiFetcher struct {
	client elasticloadbalancingv2iface.ClientAPI
}

func newAPIFetcher(clients []elasticloadbalancingv2iface.ClientAPI) fetcher {
	fetchers := make([]fetcher, len(clients))
	for idx, client := range clients {
		fetchers[idx] = &apiFetcher{client}
	}
	return &apiMultiFetcher{fetchers}
}

// fetch attempts to request the full list of lbListener objects.
// It accomplishes this by fetching a page of load balancers, then one go routine
// per listener API request. Each page of results has O(n)+1 perf since we need that
// additional fetch per lb. We let the goroutine scheduler sort things out, and use
// a sync.Pool to limit the number of in-flight requests.
func (f *apiFetcher) fetch(ctx context.Context) ([]*lbListener, error) {
	var pageSize int64 = 50

	req := f.client.DescribeLoadBalancersRequest(&elasticloadbalancingv2.DescribeLoadBalancersInput{PageSize: &pageSize})

	ctx, cancel := context.WithCancel(ctx)
	ir := &fetchRequest{
		paginator: elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(req),
		client:    f.client,
		taskPool:  sync.Pool{},
		context:   ctx,
		cancel:    cancel,
		logger:    logp.NewLogger("autodiscover-elb-fetch"),
	}

	// Limit concurrency against the AWS API by creating a pool of objects
	// This is hard coded for now. The concurrency limit of 10 was set semi-arbitrarily.
	for i := 0; i < 10; i++ {
		ir.taskPool.Put(nil)
	}

	return ir.fetch()
}

// fetchRequest provides a way to get all pages from a
// elbv2.DescribeLoadBalancersPager and all listeners for the given LoadBalancers.
type fetchRequest struct {
	paginator    elasticloadbalancingv2.DescribeLoadBalancersPaginator
	client       elasticloadbalancingv2iface.ClientAPI
	lbListeners  []*lbListener
	errs         []error
	resultsLock  sync.Mutex
	taskPool     sync.Pool
	pendingTasks sync.WaitGroup
	context      context.Context
	cancel       func()
	logger       *logp.Logger
}

func (p *fetchRequest) fetch() ([]*lbListener, error) {
	p.dispatch(p.fetchAllPages)

	// Only fetch future pages when there are no longer requests in-flight from a previous page
	p.pendingTasks.Wait()

	// Acquire the results lock to ensure memory
	// consistency between the last write and this read
	p.resultsLock.Lock()
	defer p.resultsLock.Unlock()

	// Since everything is async we have to retrieve any errors that occurred from here
	if len(p.errs) > 0 {
		return nil, multierr.Combine(p.errs...)
	}

	return p.lbListeners, nil
}

func (p *fetchRequest) fetchAllPages() {
	// Keep fetching pages unless we're stopped OR there are no pages left
	for {
		select {
		case <-p.context.Done():
			p.logger.Debug("done fetching ELB pages, context cancelled")
			return
		default:
			if !p.fetchNextPage() {
				p.logger.Debug("fetched all ELB pages")
				return
			}
			p.logger.Debug("fetched ELB page")
		}
	}
}

func (p *fetchRequest) fetchNextPage() (more bool) {
	success := p.paginator.Next(p.context)

	if success {
		for _, lb := range p.paginator.CurrentPage().LoadBalancers {
			p.dispatch(func() { p.fetchListeners(lb) })
		}
	}

	if p.paginator.Err() != nil {
		p.recordErrResult(p.paginator.Err())
	}

	return success
}

// dispatch runs the given func in a new goroutine, properly throttling requests
// with the taskPool and also managing the pendingTasks waitGroup to ensure all
// results are accumulated.
func (p *fetchRequest) dispatch(fn func()) {
	p.pendingTasks.Add(1)

	go func() {
		slot := p.taskPool.Get()
		defer p.taskPool.Put(slot)
		defer p.pendingTasks.Done()

		fn()
	}()
}

func (p *fetchRequest) fetchListeners(lb elasticloadbalancingv2.LoadBalancer) {
	listenReq := p.client.DescribeListenersRequest(&elasticloadbalancingv2.DescribeListenersInput{LoadBalancerArn: lb.LoadBalancerArn})
	listen := elasticloadbalancingv2.NewDescribeListenersPaginator(listenReq)

	if listen.Err() != nil {
		p.recordErrResult(listen.Err())
	}

	for {
		select {
		case <-p.context.Done():
			return
		default:
			if !listen.Next(p.context) {
				return
			}

			for _, listener := range listen.CurrentPage().Listeners {
				p.recordGoodResult(&lb, &listener)
			}
		}

	}
}

func (p *fetchRequest) recordGoodResult(lb *elasticloadbalancingv2.LoadBalancer, lbl *elasticloadbalancingv2.Listener) {
	p.resultsLock.Lock()
	defer p.resultsLock.Unlock()

	p.lbListeners = append(p.lbListeners, &lbListener{lb, lbl})
}

func (p *fetchRequest) recordErrResult(err error) {
	p.resultsLock.Lock()
	defer p.resultsLock.Unlock()

	p.errs = append(p.errs, err)

	p.cancel()
}
