package cloudformation

// AWSServerlessFunction_KinesisEvent AWS CloudFormation Resource (AWS::Serverless::Function.KinesisEvent)
// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#kinesis
type AWSServerlessFunction_KinesisEvent struct {

	// BatchSize AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#kinesis
	BatchSize int `json:"BatchSize,omitempty"`

	// StartingPosition AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#kinesis
	StartingPosition string `json:"StartingPosition,omitempty"`

	// Stream AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#kinesis
	Stream string `json:"Stream,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSServerlessFunction_KinesisEvent) AWSCloudFormationType() string {
	return "AWS::Serverless::Function.KinesisEvent"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSServerlessFunction_KinesisEvent) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
