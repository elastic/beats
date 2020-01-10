package serverless

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Function_S3Event AWS CloudFormation Resource (AWS::Serverless::Function.S3Event)
// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#s3
type Function_S3Event struct {

	// Bucket AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#s3
	Bucket string `json:"Bucket,omitempty"`

	// Events AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#s3
	Events *Function_Events `json:"Events,omitempty"`

	// Filter AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#s3
	Filter *Function_S3NotificationFilter `json:"Filter,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Function_S3Event) AWSCloudFormationType() string {
	return "AWS::Serverless::Function.S3Event"
}
