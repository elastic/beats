// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"go.uber.org/multierr"

	"github.com/elastic/beats/v7/libbeat/logp"
	awsauto "github.com/elastic/beats/v7/x-pack/libbeat/autodiscover/providers/aws"
)

// fetcher is an interface that can fetch a list of ec2Instance objects without pagination being necessary.
type fetcher interface {
	fetch(ctx context.Context) ([]*ec2Instance, error)
}

// apiMultiFetcher fetches results from multiple clients concatenating their results together
// Useful since we have a fetcher per region, this combines them.
type apiMultiFetcher struct {
	fetchers []fetcher
}

func (amf *apiMultiFetcher) fetch(ctx context.Context) ([]*ec2Instance, error) {
	fetchResults := make(chan []*ec2Instance)
	fetchErr := make(chan error)

	// Simultaneously fetch all from each region
	for _, f := range amf.fetchers {
		go func(f fetcher) {
			res, err := f.fetch(ctx)
			if err != nil {
				fetchErr <- err
			} else {
				fetchResults <- res
			}
		}(f)
	}

	var results []*ec2Instance
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
	client ec2.DescribeInstancesAPIClient
}

func newAPIFetcher(clients []ec2.DescribeInstancesAPIClient) fetcher {
	fetchers := make([]fetcher, len(clients))
	for idx, client := range clients {
		fetchers[idx] = &apiFetcher{client}
	}
	return &apiMultiFetcher{fetchers}
}

// fetch attempts to request the full list of ec2Instance objects.
// It accomplishes this by fetching a page of EC2 instances, then one go routine
// per listener API request. Each page of results has O(n)+1 perf since we need that
// additional fetch per EC2. We let the goroutine scheduler sort things out, and use
// a sync.Pool to limit the number of in-flight requests.
func (f *apiFetcher) fetch(ctx context.Context) ([]*ec2Instance, error) {
	var MaxResults int32 = 50

	describeInstanceInput := &ec2.DescribeInstancesInput{MaxResults: &MaxResults}

	svcDescribeInstances := ec2.NewDescribeInstancesPaginator(f.client, describeInstanceInput)

	ctx, cancel := context.WithCancel(ctx)
	ir := &fetchRequest{
		paginator: svcDescribeInstances,
		client:    f.client,
		taskPool:  sync.Pool{},
		context:   ctx,
		cancel:    cancel,
		logger:    logp.NewLogger("autodiscover-ec2-fetch"),
	}

	// Limit concurrency against the AWS API by creating a pool of objects
	// This is hard coded for now. The concurrency limit of 10 was set semi-arbitrarily.
	for i := 0; i < 10; i++ {
		ir.taskPool.Put(nil)
	}

	return ir.fetch()
}

// fetchRequest provides a way to get all pages from a
// ec2.DescribeInstancesPaginator and all listeners for the given EC2 instance.
type fetchRequest struct {
	paginator    *ec2.DescribeInstancesPaginator
	client       ec2.DescribeInstancesAPIClient
	ec2Instances []*ec2Instance
	errs         []error
	resultsLock  sync.Mutex
	taskPool     sync.Pool
	pendingTasks sync.WaitGroup
	context      context.Context
	cancel       func()
	logger       *logp.Logger
}

func (p *fetchRequest) fetch() ([]*ec2Instance, error) {
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

	return p.ec2Instances, nil
}

func (p *fetchRequest) fetchAllPages() {
	// Keep fetching pages unless we're stopped OR there are no pages left
	for {
		select {
		case <-p.context.Done():
			p.logger.Debug("done fetching EC2 instances, context cancelled")
			return
		default:
			if !p.paginator.HasMorePages() {
				p.logger.Debug("fetched all EC2 pages")
				return
			}
			p.fetchNextPage()
			p.logger.Debug("fetched EC2 page")
		}
	}
}

// TODO This is copy-pasted from elb/fetch.go
func (p *fetchRequest) fetchNextPage() {
	page, err := p.paginator.NextPage(p.context)
	if err != nil {
		p.recordErrResult(err)
	}

	for _, reservation := range page.Reservations {
		for _, instance := range reservation.Instances {
			p.dispatch(func() { p.fetchInstances(instance) })
		}
	}

	return
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

func (p *fetchRequest) fetchInstances(instance ec2types.Instance) {
	describeInstancesInput := &ec2.DescribeInstancesInput{InstanceIds: []string{awsauto.SafeString(instance.InstanceId)}}
	paginator := ec2.NewDescribeInstancesPaginator(p.client, describeInstancesInput)


	for {
		select {
		case <-p.context.Done():
			return
		default:
			if !paginator.HasMorePages() {
				return
			}

			page, err := paginator.NextPage(p.context)
			if err != nil {
				p.recordErrResult(err)
			}
			for _, reservation := range page.Reservations {
				for _, instance := range reservation.Instances {
					p.recordGoodResult(instance)
				}
			}
		}

	}
}

func (p *fetchRequest) recordGoodResult(instance ec2types.Instance) {
	p.resultsLock.Lock()
	defer p.resultsLock.Unlock()

	p.ec2Instances = append(p.ec2Instances, &ec2Instance{instance})
}

func (p *fetchRequest) recordErrResult(err error) {
	p.resultsLock.Lock()
	defer p.resultsLock.Unlock()

	p.errs = append(p.errs, err)

	p.cancel()
}
