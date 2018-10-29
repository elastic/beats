package cloudformation

// AWSServerlessApi_S3Location AWS CloudFormation Resource (AWS::Serverless::Api.S3Location)
// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#s3-location-object
type AWSServerlessApi_S3Location struct {

	// Bucket AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#awsserverlessfunction
	Bucket string `json:"Bucket,omitempty"`

	// Key AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#awsserverlessfunction
	Key string `json:"Key,omitempty"`

	// Version AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#awsserverlessfunction
	Version int `json:"Version,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSServerlessApi_S3Location) AWSCloudFormationType() string {
	return "AWS::Serverless::Api.S3Location"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSServerlessApi_S3Location) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
