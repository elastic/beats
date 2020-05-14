package ssm

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// MaintenanceWindowTask_TaskInvocationParameters AWS CloudFormation Resource (AWS::SSM::MaintenanceWindowTask.TaskInvocationParameters)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-taskinvocationparameters.html
type MaintenanceWindowTask_TaskInvocationParameters struct {

	// MaintenanceWindowAutomationParameters AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-taskinvocationparameters.html#cfn-ssm-maintenancewindowtask-taskinvocationparameters-maintenancewindowautomationparameters
	MaintenanceWindowAutomationParameters *MaintenanceWindowTask_MaintenanceWindowAutomationParameters `json:"MaintenanceWindowAutomationParameters,omitempty"`

	// MaintenanceWindowLambdaParameters AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-taskinvocationparameters.html#cfn-ssm-maintenancewindowtask-taskinvocationparameters-maintenancewindowlambdaparameters
	MaintenanceWindowLambdaParameters *MaintenanceWindowTask_MaintenanceWindowLambdaParameters `json:"MaintenanceWindowLambdaParameters,omitempty"`

	// MaintenanceWindowRunCommandParameters AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-taskinvocationparameters.html#cfn-ssm-maintenancewindowtask-taskinvocationparameters-maintenancewindowruncommandparameters
	MaintenanceWindowRunCommandParameters *MaintenanceWindowTask_MaintenanceWindowRunCommandParameters `json:"MaintenanceWindowRunCommandParameters,omitempty"`

	// MaintenanceWindowStepFunctionsParameters AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-taskinvocationparameters.html#cfn-ssm-maintenancewindowtask-taskinvocationparameters-maintenancewindowstepfunctionsparameters
	MaintenanceWindowStepFunctionsParameters *MaintenanceWindowTask_MaintenanceWindowStepFunctionsParameters `json:"MaintenanceWindowStepFunctionsParameters,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *MaintenanceWindowTask_TaskInvocationParameters) AWSCloudFormationType() string {
	return "AWS::SSM::MaintenanceWindowTask.TaskInvocationParameters"
}
