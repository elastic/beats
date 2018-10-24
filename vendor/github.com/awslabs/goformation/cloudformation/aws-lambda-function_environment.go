package cloudformation

// AWSLambdaFunction_Environment AWS CloudFormation Resource (AWS::Lambda::Function.Environment)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-function-environment.html
type AWSLambdaFunction_Environment struct {

	// Variables AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-function-environment.html#cfn-lambda-function-environment-variables
	Variables map[string]string `json:"Variables,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSLambdaFunction_Environment) AWSCloudFormationType() string {
	return "AWS::Lambda::Function.Environment"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSLambdaFunction_Environment) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
