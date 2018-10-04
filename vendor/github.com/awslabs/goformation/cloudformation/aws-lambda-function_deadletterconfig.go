package cloudformation

// AWSLambdaFunction_DeadLetterConfig AWS CloudFormation Resource (AWS::Lambda::Function.DeadLetterConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-function-deadletterconfig.html
type AWSLambdaFunction_DeadLetterConfig struct {

	// TargetArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-function-deadletterconfig.html#cfn-lambda-function-deadletterconfig-targetarn
	TargetArn string `json:"TargetArn,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSLambdaFunction_DeadLetterConfig) AWSCloudFormationType() string {
	return "AWS::Lambda::Function.DeadLetterConfig"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSLambdaFunction_DeadLetterConfig) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
