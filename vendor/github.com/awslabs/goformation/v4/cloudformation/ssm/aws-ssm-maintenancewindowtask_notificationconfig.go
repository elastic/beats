package ssm

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// MaintenanceWindowTask_NotificationConfig AWS CloudFormation Resource (AWS::SSM::MaintenanceWindowTask.NotificationConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-notificationconfig.html
type MaintenanceWindowTask_NotificationConfig struct {

	// NotificationArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-notificationconfig.html#cfn-ssm-maintenancewindowtask-notificationconfig-notificationarn
	NotificationArn string `json:"NotificationArn,omitempty"`

	// NotificationEvents AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-notificationconfig.html#cfn-ssm-maintenancewindowtask-notificationconfig-notificationevents
	NotificationEvents []string `json:"NotificationEvents,omitempty"`

	// NotificationType AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-notificationconfig.html#cfn-ssm-maintenancewindowtask-notificationconfig-notificationtype
	NotificationType string `json:"NotificationType,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *MaintenanceWindowTask_NotificationConfig) AWSCloudFormationType() string {
	return "AWS::SSM::MaintenanceWindowTask.NotificationConfig"
}
