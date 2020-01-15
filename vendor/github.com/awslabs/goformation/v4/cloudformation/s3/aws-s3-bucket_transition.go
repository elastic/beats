package s3

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Bucket_Transition AWS CloudFormation Resource (AWS::S3::Bucket.Transition)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-lifecycleconfig-rule-transition.html
type Bucket_Transition struct {

	// StorageClass AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-lifecycleconfig-rule-transition.html#cfn-s3-bucket-lifecycleconfig-rule-transition-storageclass
	StorageClass string `json:"StorageClass,omitempty"`

	// TransitionDate AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-lifecycleconfig-rule-transition.html#cfn-s3-bucket-lifecycleconfig-rule-transition-transitiondate
	TransitionDate string `json:"TransitionDate,omitempty"`

	// TransitionInDays AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-lifecycleconfig-rule-transition.html#cfn-s3-bucket-lifecycleconfig-rule-transition-transitionindays
	TransitionInDays int `json:"TransitionInDays,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Bucket_Transition) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.Transition"
}
