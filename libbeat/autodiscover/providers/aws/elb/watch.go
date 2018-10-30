// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package elb

import (
	"sync"
	"time"

	"go.uber.org/multierr"

	"github.com/aws/aws-sdk-go-v2/aws/external"

	"github.com/aws/aws-sdk-go-v2/service/elbv2"

	"github.com/elastic/beats/libbeat/common/atomic"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/libbeat/common"
)

func watch(
	region string,
	interval time.Duration,
	onStart func(arn string, lbListener common.MapStr),
	onStop func(arn string),
) (stop func()) {
	lbListeners := map[string]int32{}

	// To track changes we increment the 'generation' of each entry in the map.
	// If the generation hasn't changed we know that item has been deleted in amazon.
	// TODO: Determine if the above is completely true. Could an error on amazon's side
	// result in things being incomplete? Should we verify deletions with a Describe call
	// to that specific ARN?
	var newGen int32
	done := make(chan bool)
	stop = func() {
		done <- true
	}

	cfg, err := external.LoadDefaultAWSConfig()
	cfg.Region = region
	if err != nil {
		logp.Err("error querying AWS: %s", err)
	}
	client := elbv2.New(cfg)

	go func() {
		ticker := time.NewTicker(interval)

		for {
			var stopDescribe func()

			select {
			case <-done:
				ticker.Stop()
				if stopDescribe != nil {
					stopDescribe()
				}
				break
			case <-ticker.C:
				oldGen := newGen
				newGen = oldGen + 1

				fetchedLbls, err := GetAllLbls(client)
				// If a single request fails we have to skip
				// We need all the data intact
				if err != nil {
					logp.Err("error while querying AWS ELBs: %s", err)
					continue
				}

				for _, lbl := range fetchedLbls {
					uuid := lbl.uuid()
					if _, exists := lbListeners[uuid]; !exists {
						if onStart != nil {
							onStart(uuid, lbl.toMap())
						}
					}
					lbListeners[uuid] = newGen
				}

				for uuid, entryGen := range lbListeners {
					if entryGen == oldGen {
						if onStop != nil {
							onStop(uuid)
						}
					}
				}
			}

		}
	}()

	return stop
}

type individualResult struct {
	lbListener *lbListener
	err        error
}

type inventoryRequest struct {
	paginator    elbv2.DescribeLoadBalancersPager
	elbv2Client  *elbv2.ELBV2
	running      atomic.Bool
	lbListeners  []*lbListener
	errs         []error
	resultsLock  sync.Mutex
	taskPool     sync.Pool
	pendingTasks sync.WaitGroup
}

func (p *inventoryRequest) recordGoodResult(lb *elbv2.LoadBalancer, lbl *elbv2.Listener) {
	p.resultsLock.Lock()
	defer p.resultsLock.Unlock()

	p.lbListeners = append(p.lbListeners, &lbListener{lb, lbl})
}

func (p *inventoryRequest) recordErrResult(err error) {
	p.resultsLock.Lock()
	defer p.resultsLock.Unlock()

	p.errs = append(p.errs, err)

	// Try to stop execution early
	p.running.Store(false)
}

func (p *inventoryRequest) dispatch(fn func()) {
	p.pendingTasks.Add(1)

	go func() {
		slot := p.taskPool.Get()
		defer p.taskPool.Put(slot)
		defer p.pendingTasks.Done()

		fn()
	}()
}

func (p *inventoryRequest) fetchListeners(lb elbv2.LoadBalancer) {
	listenReq := p.elbv2Client.DescribeListenersRequest(&elbv2.DescribeListenersInput{LoadBalancerArn: lb.LoadBalancerArn})
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

func (p *inventoryRequest) fetchNextPage() {
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

func (p *inventoryRequest) fetch() ([]*lbListener, error) {
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

func GetAllLbls(client *elbv2.ELBV2) ([]*lbListener, error) {
	var pageSize int64 = 50
	req := client.DescribeLoadBalancersRequest(&elbv2.DescribeLoadBalancersInput{PageSize: &pageSize})

	// Limit concurrency against the AWS API to 5
	taskPool := sync.Pool{}
	for i := 0; i < 5; i++ {
		taskPool.Put(nil)
	}

	ir := &inventoryRequest{
		req.Paginate(),
		client,
		atomic.MakeBool(true),
		[]*lbListener{},
		[]error{},
		sync.Mutex{},
		taskPool,
		sync.WaitGroup{},
	}

	return ir.fetch()
}
