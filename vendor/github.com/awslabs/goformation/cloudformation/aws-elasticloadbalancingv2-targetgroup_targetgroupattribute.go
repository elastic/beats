package cloudformation

// AWSElasticLoadBalancingV2TargetGroup_TargetGroupAttribute AWS CloudFormation Resource (AWS::ElasticLoadBalancingV2::TargetGroup.TargetGroupAttribute)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-targetgroup-targetgroupattribute.html
type AWSElasticLoadBalancingV2TargetGroup_TargetGroupAttribute struct {

	// Key AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-targetgroup-targetgroupattribute.html#cfn-elasticloadbalancingv2-targetgroup-targetgroupattribute-key
	Key string `json:"Key,omitempty"`

	// Value AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-targetgroup-targetgroupattribute.html#cfn-elasticloadbalancingv2-targetgroup-targetgroupattribute-value
	Value string `json:"Value,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSElasticLoadBalancingV2TargetGroup_TargetGroupAttribute) AWSCloudFormationType() string {
	return "AWS::ElasticLoadBalancingV2::TargetGroup.TargetGroupAttribute"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSElasticLoadBalancingV2TargetGroup_TargetGroupAttribute) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
