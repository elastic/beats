package cloudformation

// AWSServerlessFunction_FunctionEnvironment AWS CloudFormation Resource (AWS::Serverless::Function.FunctionEnvironment)
// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#environment-object
type AWSServerlessFunction_FunctionEnvironment struct {

	// Variables AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#environment-object
	Variables map[string]string `json:"Variables,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSServerlessFunction_FunctionEnvironment) AWSCloudFormationType() string {
	return "AWS::Serverless::Function.FunctionEnvironment"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSServerlessFunction_FunctionEnvironment) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
