package elb

import (
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"

	"github.com/elastic/beats/libbeat/common"
)

// lbListener is a tuple type representing an elasticloadbalancingv2.Listener and its associated elasticloadbalancingv2.LoadBalancer.
type lbListener struct {
	lb       *elasticloadbalancingv2.LoadBalancer
	listener *elasticloadbalancingv2.Listener
}

// toMap converts this lbListener into the form consumed as metadata in the autodiscovery process.
func (l *lbListener) toMap() common.MapStr {
	m := common.MapStr{}

	m["arn"] = l.listener.ListenerArn
	m["host"] = *l.lb.DNSName
	m["port"] = *l.listener.Port
	m["protocol"] = l.listener.Protocol
	m["type"] = string(l.lb.Type)
	m["scheme"] = l.lb.Scheme
	m["availability_zones"] = l.azStrings()
	m["created"] = l.lb.CreatedTime
	m["state"] = l.stateMap()
	m["load_balancer_arn"] = *l.lb.LoadBalancerArn
	m["ip_address_type"] = string(l.lb.IpAddressType)
	m["security_groups"] = l.lb.SecurityGroups
	m["vpc_id"] = *l.lb.VpcId
	m["ssl_policy"] = l.listener.SslPolicy

	return m
}

// arn returns a globally unique ID. In the case of an lbListener, that would be its listenerArn.
func (l *lbListener) arn() string {
	return *l.listener.ListenerArn
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
