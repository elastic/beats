package codestarnotifications

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// NotificationRule_Target AWS CloudFormation Resource (AWS::CodeStarNotifications::NotificationRule.Target)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codestarnotifications-notificationrule-target.html
type NotificationRule_Target struct {

	// TargetAddress AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codestarnotifications-notificationrule-target.html#cfn-codestarnotifications-notificationrule-target-targetaddress
	TargetAddress string `json:"TargetAddress,omitempty"`

	// TargetType AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codestarnotifications-notificationrule-target.html#cfn-codestarnotifications-notificationrule-target-targettype
	TargetType string `json:"TargetType,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *NotificationRule_Target) AWSCloudFormationType() string {
	return "AWS::CodeStarNotifications::NotificationRule.Target"
}
