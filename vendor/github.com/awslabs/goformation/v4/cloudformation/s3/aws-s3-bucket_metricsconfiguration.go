package s3

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Bucket_MetricsConfiguration AWS CloudFormation Resource (AWS::S3::Bucket.MetricsConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-metricsconfiguration.html
type Bucket_MetricsConfiguration struct {

	// Id AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-metricsconfiguration.html#cfn-s3-bucket-metricsconfiguration-id
	Id string `json:"Id,omitempty"`

	// Prefix AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-metricsconfiguration.html#cfn-s3-bucket-metricsconfiguration-prefix
	Prefix string `json:"Prefix,omitempty"`

	// TagFilters AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-metricsconfiguration.html#cfn-s3-bucket-metricsconfiguration-tagfilters
	TagFilters []Bucket_TagFilter `json:"TagFilters,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Bucket_MetricsConfiguration) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.MetricsConfiguration"
}
