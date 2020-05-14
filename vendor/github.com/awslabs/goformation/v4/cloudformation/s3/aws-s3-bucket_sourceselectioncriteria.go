package s3

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Bucket_SourceSelectionCriteria AWS CloudFormation Resource (AWS::S3::Bucket.SourceSelectionCriteria)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-sourceselectioncriteria.html
type Bucket_SourceSelectionCriteria struct {

	// SseKmsEncryptedObjects AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-sourceselectioncriteria.html#cfn-s3-bucket-sourceselectioncriteria-ssekmsencryptedobjects
	SseKmsEncryptedObjects *Bucket_SseKmsEncryptedObjects `json:"SseKmsEncryptedObjects,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Bucket_SourceSelectionCriteria) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.SourceSelectionCriteria"
}
