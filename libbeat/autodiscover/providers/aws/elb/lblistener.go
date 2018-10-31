package elb

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/elbv2"

	"github.com/elastic/beats/libbeat/common"
)

func lblToMap(l *elbv2.LoadBalancer, listener *elbv2.Listener) {

}

type lbListener struct {
	lb       *elbv2.LoadBalancer
	listener *elbv2.Listener
}

func (l *lbListener) toMap() common.MapStr {
	m := common.MapStr{}

	m["host"] = *l.lb.DNSName
	m["port"] = *l.listener.Port
	m["type"] = string(l.lb.Type)
	m["scheme"] = l.lb.Scheme
	m["availability_zones"] = l.azStrings()
	m["created"] = l.lb.CreatedTime
	m["state"] = l.stateMap()
	m["load_balancer_arn"] = *l.lb.LoadBalancerArn
	m["ip_address_type"] = string(l.lb.IpAddressType)
	m["security_groups"] = l.lb.SecurityGroups
	m["vpc_id"] = *l.lb.VpcId
	m["protocol"] = l.listener.Protocol
	m["ssl_policy"] = l.listener.SslPolicy

	return m
}

func (l *lbListener) uuid() string {
	return fmt.Sprintf("%s|%s", *l.lb.LoadBalancerArn, *l.listener.ListenerArn)
}

func (l *lbListener) azStrings() []string {
	azs := l.lb.AvailabilityZones
	res := make([]string, 0, len(azs))
	for _, az := range azs {
		res = append(res, *az.ZoneName)
	}
	return res
}

func (l *lbListener) stateMap() (stateMap common.MapStr) {
	state := l.lb.State
	stateMap = common.MapStr{}
	if state.Reason != nil {
		stateMap["reason"] = *state.Reason
	}
	stateMap["code"] = state.Code
	return stateMap
}
