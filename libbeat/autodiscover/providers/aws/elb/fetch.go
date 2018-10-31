package elb

import (
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/elbv2"
	"go.uber.org/multierr"

	"github.com/elastic/beats/libbeat/common/atomic"
)

type Fetcher interface {
	fetch() ([]*lbListener, error)
}

type APIFetcher struct {
	client *elbv2.ELBV2
}

func NewAPIFetcher(client *elbv2.ELBV2) Fetcher {
	return &APIFetcher{client}
}

func (f *APIFetcher) fetch() ([]*lbListener, error) {
	var pageSize int64 = 50
	req := f.client.DescribeLoadBalancersRequest(&elbv2.DescribeLoadBalancersInput{PageSize: &pageSize})

	// Limit concurrency against the AWS API to 5
	taskPool := sync.Pool{}
	for i := 0; i < 5; i++ {
		taskPool.Put(nil)
	}

	ir := &fetchRequest{
		req.Paginate(),
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

type fetchRequest struct {
	paginator    elbv2.DescribeLoadBalancersPager
	client       *elbv2.ELBV2
	running      atomic.Bool
	lbListeners  []*lbListener
	errs         []error
	resultsLock  sync.Mutex
	taskPool     sync.Pool
	pendingTasks sync.WaitGroup
}

func (p *fetchRequest) fetch() ([]*lbListener, error) {
	p.dispatch(p.fetchNextPage)

	p.pendingTasks.Wait()

	// Acquire the results lock to ensure memory
	// consistency between the last write and this read
	p.resultsLock.Lock()
	defer p.resultsLock.Unlock()

	if len(p.errs) > 0 {
		return nil, multierr.Combine(p.errs...)
	}

	return p.lbListeners, nil
}

func (p *fetchRequest) fetchNextPage() {
	if !p.running.Load() {
		return
	}

	if p.paginator.Next() {
		for _, lb := range p.paginator.CurrentPage().LoadBalancers {
			p.dispatch(func() { p.fetchListeners(lb) })
		}
	}

	if p.paginator.Err() != nil {
		p.recordErrResult(p.paginator.Err())
	}
}

func (p *fetchRequest) dispatch(fn func()) {
	p.pendingTasks.Add(1)

	go func() {
		slot := p.taskPool.Get()
		defer p.taskPool.Put(slot)
		defer p.pendingTasks.Done()

		fn()
	}()
}

func (p *fetchRequest) recordGoodResult(lb *elbv2.LoadBalancer, lbl *elbv2.Listener) {
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

func (p *fetchRequest) fetchListeners(lb elbv2.LoadBalancer) {
	listenReq := p.client.DescribeListenersRequest(&elbv2.DescribeListenersInput{LoadBalancerArn: lb.LoadBalancerArn})
	listen := listenReq.Paginate()
	for listen.Next() && p.running.Load() {
		for _, listener := range listen.CurrentPage().Listeners {
			p.recordGoodResult(&lb, &listener)
		}
	}
	if listen.Err() != nil {
		p.recordErrResult(listen.Err())
	}
}
