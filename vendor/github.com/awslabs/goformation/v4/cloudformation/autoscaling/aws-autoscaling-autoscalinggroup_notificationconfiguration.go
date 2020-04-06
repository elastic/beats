package autoscaling

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// AutoScalingGroup_NotificationConfiguration AWS CloudFormation Resource (AWS::AutoScaling::AutoScalingGroup.NotificationConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-as-notificationconfigurations.html
type AutoScalingGroup_NotificationConfiguration struct {

	// NotificationTypes AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-as-notificationconfigurations.html#cfn-as-group-notificationconfigurations-notificationtypes
	NotificationTypes []string `json:"NotificationTypes,omitempty"`

	// TopicARN AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-as-notificationconfigurations.html#cfn-autoscaling-autoscalinggroup-notificationconfigurations-topicarn
	TopicARN string `json:"TopicARN,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AutoScalingGroup_NotificationConfiguration) AWSCloudFormationType() string {
	return "AWS::AutoScaling::AutoScalingGroup.NotificationConfiguration"
}
