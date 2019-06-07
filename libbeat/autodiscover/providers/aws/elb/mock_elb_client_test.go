package elb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/elasticloadbalancingv2iface"
)

func newMockELBClient(numResults int) mockELBClient {
	return mockELBClient{numResults: numResults}
}

type mockELBClient struct {
	elasticloadbalancingv2iface.ClientAPI
	numResults int
}

func (mockELBClient) AddListenerCertificatesRequest(*elasticloadbalancingv2.AddListenerCertificatesInput) elasticloadbalancingv2.AddListenerCertificatesRequest {
	panic("implement me")
}

func (mockELBClient) AddTagsRequest(*elasticloadbalancingv2.AddTagsInput) elasticloadbalancingv2.AddTagsRequest {
	panic("implement me")
}

func (mockELBClient) CreateListenerRequest(*elasticloadbalancingv2.CreateListenerInput) elasticloadbalancingv2.CreateListenerRequest {
	panic("implement me")
}

func (mockELBClient) CreateLoadBalancerRequest(*elasticloadbalancingv2.CreateLoadBalancerInput) elasticloadbalancingv2.CreateLoadBalancerRequest {
	panic("implement me")
}

func (mockELBClient) CreateRuleRequest(*elasticloadbalancingv2.CreateRuleInput) elasticloadbalancingv2.CreateRuleRequest {
	panic("implement me")
}

func (mockELBClient) CreateTargetGroupRequest(*elasticloadbalancingv2.CreateTargetGroupInput) elasticloadbalancingv2.CreateTargetGroupRequest {
	panic("implement me")
}

func (mockELBClient) DeleteListenerRequest(*elasticloadbalancingv2.DeleteListenerInput) elasticloadbalancingv2.DeleteListenerRequest {
	panic("implement me")
}

func (mockELBClient) DeleteLoadBalancerRequest(*elasticloadbalancingv2.DeleteLoadBalancerInput) elasticloadbalancingv2.DeleteLoadBalancerRequest {
	panic("implement me")
}

func (mockELBClient) DeleteRuleRequest(*elasticloadbalancingv2.DeleteRuleInput) elasticloadbalancingv2.DeleteRuleRequest {
	panic("implement me")
}

func (mockELBClient) DeleteTargetGroupRequest(*elasticloadbalancingv2.DeleteTargetGroupInput) elasticloadbalancingv2.DeleteTargetGroupRequest {
	panic("implement me")
}

func (mockELBClient) DeregisterTargetsRequest(*elasticloadbalancingv2.DeregisterTargetsInput) elasticloadbalancingv2.DeregisterTargetsRequest {
	panic("implement me")
}

func (mockELBClient) DescribeAccountLimitsRequest(*elasticloadbalancingv2.DescribeAccountLimitsInput) elasticloadbalancingv2.DescribeAccountLimitsRequest {
	panic("implement me")
}

func (mockELBClient) DescribeListenerCertificatesRequest(*elasticloadbalancingv2.DescribeListenerCertificatesInput) elasticloadbalancingv2.DescribeListenerCertificatesRequest {
	panic("implement me")
}

func (mockELBClient) DescribeListenersRequest(*elasticloadbalancingv2.DescribeListenersInput) elasticloadbalancingv2.DescribeListenersRequest {
	panic("implement me")
}

func (mockELBClient) DescribeLoadBalancerAttributesRequest(*elasticloadbalancingv2.DescribeLoadBalancerAttributesInput) elasticloadbalancingv2.DescribeLoadBalancerAttributesRequest {
	panic("implement me")
}

func (mockELBClient) DescribeLoadBalancersRequest(*elasticloadbalancingv2.DescribeLoadBalancersInput) elasticloadbalancingv2.DescribeLoadBalancersRequest {
	panic("implement me")
}

func (mockELBClient) DescribeRulesRequest(*elasticloadbalancingv2.DescribeRulesInput) elasticloadbalancingv2.DescribeRulesRequest {
	panic("implement me")
}

func (mockELBClient) DescribeSSLPoliciesRequest(*elasticloadbalancingv2.DescribeSSLPoliciesInput) elasticloadbalancingv2.DescribeSSLPoliciesRequest {
	panic("implement me")
}

func (mockELBClient) DescribeTagsRequest(*elasticloadbalancingv2.DescribeTagsInput) elasticloadbalancingv2.DescribeTagsRequest {
	panic("implement me")
}

func (mockELBClient) DescribeTargetGroupAttributesRequest(*elasticloadbalancingv2.DescribeTargetGroupAttributesInput) elasticloadbalancingv2.DescribeTargetGroupAttributesRequest {
	panic("implement me")
}

func (mockELBClient) DescribeTargetGroupsRequest(*elasticloadbalancingv2.DescribeTargetGroupsInput) elasticloadbalancingv2.DescribeTargetGroupsRequest {
	panic("implement me")
}

func (mockELBClient) DescribeTargetHealthRequest(*elasticloadbalancingv2.DescribeTargetHealthInput) elasticloadbalancingv2.DescribeTargetHealthRequest {
	panic("implement me")
}

func (mockELBClient) ModifyListenerRequest(*elasticloadbalancingv2.ModifyListenerInput) elasticloadbalancingv2.ModifyListenerRequest {
	panic("implement me")
}

func (mockELBClient) ModifyLoadBalancerAttributesRequest(*elasticloadbalancingv2.ModifyLoadBalancerAttributesInput) elasticloadbalancingv2.ModifyLoadBalancerAttributesRequest {
	panic("implement me")
}

func (mockELBClient) ModifyRuleRequest(*elasticloadbalancingv2.ModifyRuleInput) elasticloadbalancingv2.ModifyRuleRequest {
	panic("implement me")
}

func (mockELBClient) ModifyTargetGroupRequest(*elasticloadbalancingv2.ModifyTargetGroupInput) elasticloadbalancingv2.ModifyTargetGroupRequest {
	panic("implement me")
}

func (mockELBClient) ModifyTargetGroupAttributesRequest(*elasticloadbalancingv2.ModifyTargetGroupAttributesInput) elasticloadbalancingv2.ModifyTargetGroupAttributesRequest {
	panic("implement me")
}

func (mockELBClient) RegisterTargetsRequest(*elasticloadbalancingv2.RegisterTargetsInput) elasticloadbalancingv2.RegisterTargetsRequest {
	panic("implement me")
}

func (mockELBClient) RemoveListenerCertificatesRequest(*elasticloadbalancingv2.RemoveListenerCertificatesInput) elasticloadbalancingv2.RemoveListenerCertificatesRequest {
	panic("implement me")
}

func (mockELBClient) RemoveTagsRequest(*elasticloadbalancingv2.RemoveTagsInput) elasticloadbalancingv2.RemoveTagsRequest {
	panic("implement me")
}

func (mockELBClient) SetIpAddressTypeRequest(*elasticloadbalancingv2.SetIpAddressTypeInput) elasticloadbalancingv2.SetIpAddressTypeRequest {
	panic("implement me")
}

func (mockELBClient) SetRulePrioritiesRequest(*elasticloadbalancingv2.SetRulePrioritiesInput) elasticloadbalancingv2.SetRulePrioritiesRequest {
	panic("implement me")
}

func (mockELBClient) SetSecurityGroupsRequest(*elasticloadbalancingv2.SetSecurityGroupsInput) elasticloadbalancingv2.SetSecurityGroupsRequest {
	panic("implement me")
}

func (mockELBClient) SetSubnetsRequest(*elasticloadbalancingv2.SetSubnetsInput) elasticloadbalancingv2.SetSubnetsRequest {
	panic("implement me")
}

func (mockELBClient) WaitUntilLoadBalancerAvailable(context.Context, *elasticloadbalancingv2.DescribeLoadBalancersInput, ...aws.WaiterOption) error {
	panic("implement me")
}

func (mockELBClient) WaitUntilLoadBalancerExists(context.Context, *elasticloadbalancingv2.DescribeLoadBalancersInput, ...aws.WaiterOption) error {
	panic("implement me")
}

func (mockELBClient) WaitUntilLoadBalancersDeleted(context.Context, *elasticloadbalancingv2.DescribeLoadBalancersInput, ...aws.WaiterOption) error {
	panic("implement me")
}

func (mockELBClient) WaitUntilTargetDeregistered(context.Context, *elasticloadbalancingv2.DescribeTargetHealthInput, ...aws.WaiterOption) error {
	panic("implement me")
}

func (mockELBClient) WaitUntilTargetInService(context.Context, *elasticloadbalancingv2.DescribeTargetHealthInput, ...aws.WaiterOption) error {
	panic("implement me")
}
