package elb

import (
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/elbv2"
)

type mockFetcher struct {
	lblListeners []*lbListener
	err          error
	lock         sync.Mutex
}

func newMockFetcher(lbListeners []*lbListener, err error) *mockFetcher {
	return &mockFetcher{lblListeners: lbListeners, err: err}
}

func (f *mockFetcher) fetch() ([]*lbListener, error) {
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

func fakeLbl() *lbListener {
	dnsName := "fake.example.net"
	strVal := "strVal"
	lbARN := "lb_arn"
	listenerARN := "listen_arn"
	state := elbv2.LoadBalancerState{Reason: &strVal, Code: elbv2.LoadBalancerStateEnumActive}
	now := time.Now()

	lb := &elbv2.LoadBalancer{
		LoadBalancerArn:   &lbARN,
		DNSName:           &dnsName,
		Type:              elbv2.LoadBalancerTypeEnumApplication,
		Scheme:            elbv2.LoadBalancerSchemeEnumInternetFacing,
		AvailabilityZones: []elbv2.AvailabilityZone{{ZoneName: &strVal}},
		CreatedTime:       &now,
		State:             &state,
		IpAddressType:     elbv2.IpAddressTypeDualstack,
		SecurityGroups:    []string{"foo-security-group"},
		VpcId:             &strVal,
	}

	var port int64 = 1234
	listener := &elbv2.Listener{
		ListenerArn:     &listenerARN,
		LoadBalancerArn: lb.LoadBalancerArn,
		Port:            &port,
		Protocol:        elbv2.ProtocolEnumHttps,
		SslPolicy:       &strVal,
	}

	return &lbListener{lb: lb, listener: listener}
}
