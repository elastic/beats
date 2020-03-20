package s3

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Bucket_ObjectLockRule AWS CloudFormation Resource (AWS::S3::Bucket.ObjectLockRule)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-objectlockrule.html
type Bucket_ObjectLockRule struct {

	// DefaultRetention AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-objectlockrule.html#cfn-s3-bucket-objectlockrule-defaultretention
	DefaultRetention *Bucket_DefaultRetention `json:"DefaultRetention,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Bucket_ObjectLockRule) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.ObjectLockRule"
}
