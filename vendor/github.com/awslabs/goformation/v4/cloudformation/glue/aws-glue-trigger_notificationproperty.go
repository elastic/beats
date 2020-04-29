package glue

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Trigger_NotificationProperty AWS CloudFormation Resource (AWS::Glue::Trigger.NotificationProperty)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-trigger-notificationproperty.html
type Trigger_NotificationProperty struct {

	// NotifyDelayAfter AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-trigger-notificationproperty.html#cfn-glue-trigger-notificationproperty-notifydelayafter
	NotifyDelayAfter int `json:"NotifyDelayAfter,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Trigger_NotificationProperty) AWSCloudFormationType() string {
	return "AWS::Glue::Trigger.NotificationProperty"
}
