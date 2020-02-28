package s3

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Bucket_CorsConfiguration AWS CloudFormation Resource (AWS::S3::Bucket.CorsConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-cors.html
type Bucket_CorsConfiguration struct {

	// CorsRules AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-cors.html#cfn-s3-bucket-cors-corsrule
	CorsRules []Bucket_CorsRule `json:"CorsRules,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Bucket_CorsConfiguration) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.CorsConfiguration"
}
