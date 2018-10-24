package cloudformation

// AWSElasticLoadBalancingV2LoadBalancer_SubnetMapping AWS CloudFormation Resource (AWS::ElasticLoadBalancingV2::LoadBalancer.SubnetMapping)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-loadbalancer-subnetmapping.html
type AWSElasticLoadBalancingV2LoadBalancer_SubnetMapping struct {

	// AllocationId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-loadbalancer-subnetmapping.html#cfn-elasticloadbalancingv2-loadbalancer-subnetmapping-allocationid
	AllocationId string `json:"AllocationId,omitempty"`

	// SubnetId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-loadbalancer-subnetmapping.html#cfn-elasticloadbalancingv2-loadbalancer-subnetmapping-subnetid
	SubnetId string `json:"SubnetId,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSElasticLoadBalancingV2LoadBalancer_SubnetMapping) AWSCloudFormationType() string {
	return "AWS::ElasticLoadBalancingV2::LoadBalancer.SubnetMapping"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSElasticLoadBalancingV2LoadBalancer_SubnetMapping) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
