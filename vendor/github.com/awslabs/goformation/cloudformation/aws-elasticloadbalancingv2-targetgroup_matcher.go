package cloudformation

// AWSElasticLoadBalancingV2TargetGroup_Matcher AWS CloudFormation Resource (AWS::ElasticLoadBalancingV2::TargetGroup.Matcher)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-targetgroup-matcher.html
type AWSElasticLoadBalancingV2TargetGroup_Matcher struct {

	// HttpCode AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-targetgroup-matcher.html#cfn-elasticloadbalancingv2-targetgroup-matcher-httpcode
	HttpCode string `json:"HttpCode,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSElasticLoadBalancingV2TargetGroup_Matcher) AWSCloudFormationType() string {
	return "AWS::ElasticLoadBalancingV2::TargetGroup.Matcher"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSElasticLoadBalancingV2TargetGroup_Matcher) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
