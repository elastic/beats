package stepfunctions

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// StateMachine_LogDestination AWS CloudFormation Resource (AWS::StepFunctions::StateMachine.LogDestination)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-stepfunctions-statemachine-logdestination.html
type StateMachine_LogDestination struct {

	// CloudWatchLogsLogGroup AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-stepfunctions-statemachine-logdestination.html#cfn-stepfunctions-statemachine-logdestination-cloudwatchlogsloggroup
	CloudWatchLogsLogGroup *StateMachine_CloudWatchLogsLogGroup `json:"CloudWatchLogsLogGroup,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *StateMachine_LogDestination) AWSCloudFormationType() string {
	return "AWS::StepFunctions::StateMachine.LogDestination"
}
