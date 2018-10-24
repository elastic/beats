package cloudformation

// AWSS3Bucket_RoutingRule AWS CloudFormation Resource (AWS::S3::Bucket.RoutingRule)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-websiteconfiguration-routingrules.html
type AWSS3Bucket_RoutingRule struct {

	// RedirectRule AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-websiteconfiguration-routingrules.html#cfn-s3-websiteconfiguration-routingrules-redirectrule
	RedirectRule *AWSS3Bucket_RedirectRule `json:"RedirectRule,omitempty"`

	// RoutingRuleCondition AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-websiteconfiguration-routingrules.html#cfn-s3-websiteconfiguration-routingrules-routingrulecondition
	RoutingRuleCondition *AWSS3Bucket_RoutingRuleCondition `json:"RoutingRuleCondition,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSS3Bucket_RoutingRule) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.RoutingRule"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSS3Bucket_RoutingRule) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
