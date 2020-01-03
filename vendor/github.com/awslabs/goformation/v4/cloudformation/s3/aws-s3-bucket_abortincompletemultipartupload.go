package s3

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Bucket_AbortIncompleteMultipartUpload AWS CloudFormation Resource (AWS::S3::Bucket.AbortIncompleteMultipartUpload)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-abortincompletemultipartupload.html
type Bucket_AbortIncompleteMultipartUpload struct {

	// DaysAfterInitiation AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-abortincompletemultipartupload.html#cfn-s3-bucket-abortincompletemultipartupload-daysafterinitiation
	DaysAfterInitiation int `json:"DaysAfterInitiation"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Bucket_AbortIncompleteMultipartUpload) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.AbortIncompleteMultipartUpload"
}
