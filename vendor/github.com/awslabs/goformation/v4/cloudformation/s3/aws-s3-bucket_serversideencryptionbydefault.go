package s3

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Bucket_ServerSideEncryptionByDefault AWS CloudFormation Resource (AWS::S3::Bucket.ServerSideEncryptionByDefault)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-serversideencryptionbydefault.html
type Bucket_ServerSideEncryptionByDefault struct {

	// KMSMasterKeyID AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-serversideencryptionbydefault.html#cfn-s3-bucket-serversideencryptionbydefault-kmsmasterkeyid
	KMSMasterKeyID string `json:"KMSMasterKeyID,omitempty"`

	// SSEAlgorithm AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-serversideencryptionbydefault.html#cfn-s3-bucket-serversideencryptionbydefault-ssealgorithm
	SSEAlgorithm string `json:"SSEAlgorithm,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Bucket_ServerSideEncryptionByDefault) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.ServerSideEncryptionByDefault"
}
