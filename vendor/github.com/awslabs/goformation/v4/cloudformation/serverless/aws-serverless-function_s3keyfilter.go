package serverless

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Function_S3KeyFilter AWS CloudFormation Resource (AWS::Serverless::Function.S3KeyFilter)
// See: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-notificationconfiguration-config-filter.html
type Function_S3KeyFilter struct {

	// Rules AWS CloudFormation Property
	// Required: true
	// See: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-notificationconfiguration-config-filter.html
	Rules []Function_S3KeyFilterRule `json:"Rules,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Function_S3KeyFilter) AWSCloudFormationType() string {
	return "AWS::Serverless::Function.S3KeyFilter"
}
