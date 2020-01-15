package s3

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Bucket_TopicConfiguration AWS CloudFormation Resource (AWS::S3::Bucket.TopicConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-notificationconfig-topicconfig.html
type Bucket_TopicConfiguration struct {

	// Event AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-notificationconfig-topicconfig.html#cfn-s3-bucket-notificationconfig-topicconfig-event
	Event string `json:"Event,omitempty"`

	// Filter AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-notificationconfig-topicconfig.html#cfn-s3-bucket-notificationconfig-topicconfig-filter
	Filter *Bucket_NotificationFilter `json:"Filter,omitempty"`

	// Topic AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-notificationconfig-topicconfig.html#cfn-s3-bucket-notificationconfig-topicconfig-topic
	Topic string `json:"Topic,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Bucket_TopicConfiguration) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.TopicConfiguration"
}
