package cloudformation

// AWSWAFWebACL_ActivatedRule AWS CloudFormation Resource (AWS::WAF::WebACL.ActivatedRule)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-webacl-rules.html
type AWSWAFWebACL_ActivatedRule struct {

	// Action AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-webacl-rules.html#cfn-waf-webacl-rules-action
	Action *AWSWAFWebACL_WafAction `json:"Action,omitempty"`

	// Priority AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-webacl-rules.html#cfn-waf-webacl-rules-priority
	Priority int `json:"Priority,omitempty"`

	// RuleId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-webacl-rules.html#cfn-waf-webacl-rules-ruleid
	RuleId string `json:"RuleId,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSWAFWebACL_ActivatedRule) AWSCloudFormationType() string {
	return "AWS::WAF::WebACL.ActivatedRule"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSWAFWebACL_ActivatedRule) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
