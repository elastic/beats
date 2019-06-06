package elb

import (
	"context"
	"sync"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"go.uber.org/multierr"

	"github.com/elastic/beats/libbeat/common/atomic"
)

// fetcher is an interface that can fetch a full list of lbListener (load balancer + listener) objects.
// representing the current state of the API.
type fetcher interface {
	fetch() ([]*lbListener, error)
}

// apiFetcher is a concrete implementation of fetcher that hits the real AWS API.
type apiFetcher struct {
	client *elasticloadbalancingv2.Client
}

func newAPIFetcher(client *elasticloadbalancingv2.Client) fetcher {
	return &apiFetcher{client}
}

// fetch attempts to request the full list of lbListener objects.
// It accomplishes this by fetching a page of load balancers, then one go routine
// per listener API request. Each page of results has O(n)+1 perf since we need that
// additional fetch per lb. We let the goroutine scheduler sort things out, and use
// a sync.Pool to limit the number of in-flight requests.
func (f *apiFetcher) fetch() ([]*lbListener, error) {
	var pageSize int64 = 50

	req := f.client.DescribeLoadBalancersRequest(&elasticloadbalancingv2.DescribeLoadBalancersInput{PageSize: &pageSize})

	// Limit concurrency against the AWS API by creating a pool of objects
	// This is hard coded for now. The concurrency limit of 10 was set semi-arbitrarily.
	taskPool := sync.Pool{}
	for i := 0; i < 10; i++ {
		taskPool.Put(nil)
	}
	ir := &fetchRequest{
		elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(req),
		f.client,
		atomic.MakeBool(true),
		[]*lbListener{},
		[]error{},
		sync.Mutex{},
		taskPool,
		sync.WaitGroup{},
	}

	return ir.fetch()
}

// fetchRequest provides a way to get all pages from a
// elbv2.DescribeLoadBalancersPager and all listeners for the given LoadBalancers.
type fetchRequest struct {
	paginator    elasticloadbalancingv2.DescribeLoadBalancersPaginator
	client       *elasticloadbalancingv2.Client
	running      atomic.Bool
	lbListeners  []*lbListener
	errs         []error
	resultsLock  sync.Mutex
	taskPool     sync.Pool
	pendingTasks sync.WaitGroup
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
	for p.running.Load() && p.fetchNextPage() {
		logp.Debug("autodiscover-elb", "API page fetched")
	}
}

func (p *fetchRequest) fetchNextPage() (more bool) {
	success := p.paginator.Next(context.TODO())

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
	for listen.Next(context.TODO()) && p.running.Load() {
		for _, listener := range listen.CurrentPage().Listeners {
			p.recordGoodResult(&lb, &listener)
		}
	}
	if listen.Err() != nil {
		p.recordErrResult(listen.Err())
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

	// Try to stop execution early
	p.running.Store(false)
}
