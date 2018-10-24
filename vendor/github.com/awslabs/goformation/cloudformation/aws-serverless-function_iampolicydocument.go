package cloudformation

// AWSServerlessFunction_IAMPolicyDocument AWS CloudFormation Resource (AWS::Serverless::Function.IAMPolicyDocument)
// See: http://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies.html
type AWSServerlessFunction_IAMPolicyDocument struct {

	// Statement AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies.html
	Statement interface{} `json:"Statement,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSServerlessFunction_IAMPolicyDocument) AWSCloudFormationType() string {
	return "AWS::Serverless::Function.IAMPolicyDocument"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSServerlessFunction_IAMPolicyDocument) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
