package cloudformation

// AWSElasticLoadBalancingV2ListenerRule_Action AWS CloudFormation Resource (AWS::ElasticLoadBalancingV2::ListenerRule.Action)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-listenerrule-actions.html
type AWSElasticLoadBalancingV2ListenerRule_Action struct {

	// TargetGroupArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-listenerrule-actions.html#cfn-elasticloadbalancingv2-listener-actions-targetgrouparn
	TargetGroupArn string `json:"TargetGroupArn,omitempty"`

	// Type AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-listenerrule-actions.html#cfn-elasticloadbalancingv2-listener-actions-type
	Type string `json:"Type,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSElasticLoadBalancingV2ListenerRule_Action) AWSCloudFormationType() string {
	return "AWS::ElasticLoadBalancingV2::ListenerRule.Action"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSElasticLoadBalancingV2ListenerRule_Action) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
