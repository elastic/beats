package sagemaker

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Workteam_NotificationConfiguration AWS CloudFormation Resource (AWS::SageMaker::Workteam.NotificationConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-sagemaker-workteam-notificationconfiguration.html
type Workteam_NotificationConfiguration struct {

	// NotificationTopicArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-sagemaker-workteam-notificationconfiguration.html#cfn-sagemaker-workteam-notificationconfiguration-notificationtopicarn
	NotificationTopicArn string `json:"NotificationTopicArn,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Workteam_NotificationConfiguration) AWSCloudFormationType() string {
	return "AWS::SageMaker::Workteam.NotificationConfiguration"
}
