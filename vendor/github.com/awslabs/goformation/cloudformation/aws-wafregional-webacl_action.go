package cloudformation

// AWSWAFRegionalWebACL_Action AWS CloudFormation Resource (AWS::WAFRegional::WebACL.Action)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafregional-webacl-action.html
type AWSWAFRegionalWebACL_Action struct {

	// Type AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafregional-webacl-action.html#cfn-wafregional-webacl-action-type
	Type string `json:"Type,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSWAFRegionalWebACL_Action) AWSCloudFormationType() string {
	return "AWS::WAFRegional::WebACL.Action"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSWAFRegionalWebACL_Action) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
