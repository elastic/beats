package glue

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Job_NotificationProperty AWS CloudFormation Resource (AWS::Glue::Job.NotificationProperty)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-job-notificationproperty.html
type Job_NotificationProperty struct {

	// NotifyDelayAfter AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-job-notificationproperty.html#cfn-glue-job-notificationproperty-notifydelayafter
	NotifyDelayAfter int `json:"NotifyDelayAfter,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Job_NotificationProperty) AWSCloudFormationType() string {
	return "AWS::Glue::Job.NotificationProperty"
}
