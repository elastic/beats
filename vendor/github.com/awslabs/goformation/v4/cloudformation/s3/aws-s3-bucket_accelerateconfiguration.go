package s3

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Bucket_AccelerateConfiguration AWS CloudFormation Resource (AWS::S3::Bucket.AccelerateConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-accelerateconfiguration.html
type Bucket_AccelerateConfiguration struct {

	// AccelerationStatus AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-accelerateconfiguration.html#cfn-s3-bucket-accelerateconfiguration-accelerationstatus
	AccelerationStatus string `json:"AccelerationStatus,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Bucket_AccelerateConfiguration) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.AccelerateConfiguration"
}
