package s3

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Bucket_ObjectLockConfiguration AWS CloudFormation Resource (AWS::S3::Bucket.ObjectLockConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-objectlockconfiguration.html
type Bucket_ObjectLockConfiguration struct {

	// ObjectLockEnabled AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-objectlockconfiguration.html#cfn-s3-bucket-objectlockconfiguration-objectlockenabled
	ObjectLockEnabled string `json:"ObjectLockEnabled,omitempty"`

	// Rule AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-objectlockconfiguration.html#cfn-s3-bucket-objectlockconfiguration-rule
	Rule *Bucket_ObjectLockRule `json:"Rule,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Bucket_ObjectLockConfiguration) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.ObjectLockConfiguration"
}
