package cloudformation

// AWSIAMRole_Policy AWS CloudFormation Resource (AWS::IAM::Role.Policy)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iam-policy.html
type AWSIAMRole_Policy struct {

	// PolicyDocument AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iam-policy.html#cfn-iam-policies-policydocument
	PolicyDocument interface{} `json:"PolicyDocument,omitempty"`

	// PolicyName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iam-policy.html#cfn-iam-policies-policyname
	PolicyName string `json:"PolicyName,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSIAMRole_Policy) AWSCloudFormationType() string {
	return "AWS::IAM::Role.Policy"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSIAMRole_Policy) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
