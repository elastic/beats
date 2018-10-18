package cloudformation

// AWSServerlessFunction_S3Event AWS CloudFormation Resource (AWS::Serverless::Function.S3Event)
// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#s3
type AWSServerlessFunction_S3Event struct {

	// Bucket AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#s3
	Bucket string `json:"Bucket,omitempty"`

	// Events AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#s3
	Events *AWSServerlessFunction_Events `json:"Events,omitempty"`

	// Filter AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#s3
	Filter *AWSServerlessFunction_S3NotificationFilter `json:"Filter,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSServerlessFunction_S3Event) AWSCloudFormationType() string {
	return "AWS::Serverless::Function.S3Event"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSServerlessFunction_S3Event) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
