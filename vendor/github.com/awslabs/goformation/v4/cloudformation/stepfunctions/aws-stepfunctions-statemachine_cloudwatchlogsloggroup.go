package stepfunctions

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// StateMachine_CloudWatchLogsLogGroup AWS CloudFormation Resource (AWS::StepFunctions::StateMachine.CloudWatchLogsLogGroup)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-stepfunctions-statemachine-logdestination-cloudwatchlogsloggroup.html
type StateMachine_CloudWatchLogsLogGroup struct {

	// LogGroupArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-stepfunctions-statemachine-logdestination-cloudwatchlogsloggroup.html#cfn-stepfunctions-statemachine-logdestination-cloudwatchlogsloggroup-loggrouparn
	LogGroupArn string `json:"LogGroupArn,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *StateMachine_CloudWatchLogsLogGroup) AWSCloudFormationType() string {
	return "AWS::StepFunctions::StateMachine.CloudWatchLogsLogGroup"
}
