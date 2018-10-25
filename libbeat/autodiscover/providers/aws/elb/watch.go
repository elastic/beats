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
	"fmt"
	"time"

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

				var err error
				stopDescribe, err = describeEachLBListener(region, func(lbl lbListener) {
					uuid := lbl.uuid()
					if _, exists := lbListeners[uuid]; !exists {
						if onStart != nil {
							onStart(uuid, lbl.toMap())
						}
					}
					lbListeners[uuid] = newGen
				})

				if err != nil {
					logp.Err("error while querying AWS ELBs: %s", err)
					continue
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

func describeEachLBListener(region string, cb func(lbl lbListener)) (stop func(), err error) {
	cfg, err := external.LoadDefaultAWSConfig()
	cfg.Region = region
	if err != nil {
		return nil, err
	}
	e := elbv2.New(cfg)

	running := atomic.NewBool(true)
	stop = func() {
		running.Store(false)
	}

	var pageSize int64 = 100
	describe := e.DescribeLoadBalancersRequest(&elbv2.DescribeLoadBalancersInput{PageSize: &pageSize}).Paginate()

	go func() {
		for describe.Next() && running.Load() {
			for _, lb := range describe.CurrentPage().LoadBalancers {
				go func() {
					listen := e.DescribeListenersRequest(&elbv2.DescribeListenersInput{LoadBalancerArn: lb.LoadBalancerArn}).Paginate()
					for listen.Next() && running.Load() {
						for _, listener := range listen.CurrentPage().Listeners {
							lbl := lbListener{&lb, &listener}
							cb(lbl)
						}
					}
					if err = listen.Err(); err != nil {
						logp.Err(fmt.Sprintf("Could not describe load balancer listeners: %s", err))
						return
					}
				}()
			}
		}
	}()

	if err := describe.Err(); err != nil {
		return stop, err
	}

	return stop, err
}
