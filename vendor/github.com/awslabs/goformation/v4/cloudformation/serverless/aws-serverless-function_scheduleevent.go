package serverless

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Function_ScheduleEvent AWS CloudFormation Resource (AWS::Serverless::Function.ScheduleEvent)
// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#schedule
type Function_ScheduleEvent struct {

	// Input AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#schedule
	Input string `json:"Input,omitempty"`

	// Schedule AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#schedule
	Schedule string `json:"Schedule,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Function_ScheduleEvent) AWSCloudFormationType() string {
	return "AWS::Serverless::Function.ScheduleEvent"
}
