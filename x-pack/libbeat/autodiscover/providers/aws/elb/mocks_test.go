// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elb

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

// mockFetcher is a fetcher that returns a customizable list of results, useful for testing.
type mockFetcher struct {
	lblListeners []*lbListener
	err          error
	lock         sync.Mutex
}

func newMockFetcher(lbListeners []*lbListener, err error) *mockFetcher {
	return &mockFetcher{lblListeners: lbListeners, err: err}
}

func (f *mockFetcher) fetch(ctx context.Context) ([]*lbListener, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	result := make([]*lbListener, len(f.lblListeners))
	copy(result, f.lblListeners)

	return result, f.err
}

func (f *mockFetcher) setLbls(newLbls []*lbListener) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.lblListeners = newLbls
}

func (f *mockFetcher) setError(err error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.lblListeners = []*lbListener{}
	f.err = err
}

func fakeLbl() *lbListener {
	dnsName := "fake.example.net"
	strVal := "strVal"
	lbARN := "lb_arn"
	listenerARN := "listen_arn"
	state := elasticloadbalancingv2.LoadBalancerState{Reason: &strVal, Code: elasticloadbalancingv2.LoadBalancerStateEnumActive}
	now := time.Now()

	lb := &elasticloadbalancingv2.LoadBalancer{
		LoadBalancerArn:   &lbARN,
		DNSName:           &dnsName,
		Type:              elasticloadbalancingv2.LoadBalancerTypeEnumApplication,
		Scheme:            elasticloadbalancingv2.LoadBalancerSchemeEnumInternetFacing,
		AvailabilityZones: []elasticloadbalancingv2.AvailabilityZone{{ZoneName: &strVal}},
		CreatedTime:       &now,
		State:             &state,
		IpAddressType:     elasticloadbalancingv2.IpAddressTypeDualstack,
		SecurityGroups:    []string{"foo-security-group"},
		VpcId:             &strVal,
	}

	var port int64 = 1234
	listener := &elasticloadbalancingv2.Listener{
		ListenerArn:     &listenerARN,
		LoadBalancerArn: lb.LoadBalancerArn,
		Port:            &port,
		Protocol:        elasticloadbalancingv2.ProtocolEnumHttps,
		SslPolicy:       &strVal,
	}

	return &lbListener{lb: lb, listener: listener}
}
