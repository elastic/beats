// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elb

import (
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"

	"github.com/elastic/beats/v8/libbeat/common"
	awsauto "github.com/elastic/beats/v8/x-pack/libbeat/autodiscover/providers/aws"
)

// lbListener is a tuple type representing an elasticloadbalancingv2.Listener and its associated elasticloadbalancingv2.LoadBalancer.
type lbListener struct {
	lb       *elasticloadbalancingv2.LoadBalancer
	listener *elasticloadbalancingv2.Listener
}

// toMap converts this lbListener into the form consumed as metadata in the autodiscovery process.
func (l *lbListener) toMap() common.MapStr {
	// We fully spell out listener_arn to avoid confusion with the ARN for the whole ELB
	m := common.MapStr{
		"listener_arn":       l.listener.ListenerArn,
		"load_balancer_arn":  awsauto.SafeString(l.lb.LoadBalancerArn),
		"host":               awsauto.SafeString(l.lb.DNSName),
		"protocol":           l.listener.Protocol,
		"type":               string(l.lb.Type),
		"scheme":             l.lb.Scheme,
		"availability_zones": l.azStrings(),
		"created":            l.lb.CreatedTime,
		"state":              l.stateMap(),
		"ip_address_type":    string(l.lb.IpAddressType),
		"security_groups":    l.lb.SecurityGroups,
		"vpc_id":             awsauto.SafeString(l.lb.VpcId),
		"ssl_policy":         l.listener.SslPolicy,
	}

	if l.listener.Port != nil {
		m["port"] = *l.listener.Port
	}

	return m
}

func (l *lbListener) toCloudMap() common.MapStr {
	m := common.MapStr{}

	var azs []string
	for _, az := range l.lb.AvailabilityZones {
		azs = append(azs, *az.ZoneName)
	}
	m["availability_zone"] = azs
	m["provider"] = "aws"

	// The region is just an AZ with the last character removed
	firstAz := azs[0]
	m["region"] = firstAz[:len(firstAz)-2]

	return m
}

// arn returns a globally unique ID. In the case of an lbListener, that would be its listenerArn.
func (l *lbListener) arn() string {
	return *l.listener.ListenerArn
}

// azStrings transforms the weird list of availability zone string pointers to a slice of plain strings.
func (l *lbListener) azStrings() []string {
	azs := l.lb.AvailabilityZones
	res := make([]string, 0, len(azs))
	for _, az := range azs {
		res = append(res, *az.ZoneName)
	}
	return res
}

// stateMap converts the State part of the lb struct into a friendlier map with 'reason' and 'code' fields.
func (l *lbListener) stateMap() (stateMap common.MapStr) {
	state := l.lb.State
	stateMap = common.MapStr{}
	if state.Reason != nil {
		stateMap["reason"] = *state.Reason
	}
	stateMap["code"] = state.Code
	return stateMap
}
