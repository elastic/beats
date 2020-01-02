package s3

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Bucket_StorageClassAnalysis AWS CloudFormation Resource (AWS::S3::Bucket.StorageClassAnalysis)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-storageclassanalysis.html
type Bucket_StorageClassAnalysis struct {

	// DataExport AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-storageclassanalysis.html#cfn-s3-bucket-storageclassanalysis-dataexport
	DataExport *Bucket_DataExport `json:"DataExport,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Bucket_StorageClassAnalysis) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.StorageClassAnalysis"
}
