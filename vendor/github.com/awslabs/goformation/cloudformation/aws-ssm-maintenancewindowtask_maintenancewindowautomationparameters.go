package cloudformation

// AWSSSMMaintenanceWindowTask_MaintenanceWindowAutomationParameters AWS CloudFormation Resource (AWS::SSM::MaintenanceWindowTask.MaintenanceWindowAutomationParameters)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-maintenancewindowautomationparameters.html
type AWSSSMMaintenanceWindowTask_MaintenanceWindowAutomationParameters struct {

	// DocumentVersion AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-maintenancewindowautomationparameters.html#cfn-ssm-maintenancewindowtask-maintenancewindowautomationparameters-documentversion
	DocumentVersion string `json:"DocumentVersion,omitempty"`

	// Parameters AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-maintenancewindowautomationparameters.html#cfn-ssm-maintenancewindowtask-maintenancewindowautomationparameters-parameters
	Parameters interface{} `json:"Parameters,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSSMMaintenanceWindowTask_MaintenanceWindowAutomationParameters) AWSCloudFormationType() string {
	return "AWS::SSM::MaintenanceWindowTask.MaintenanceWindowAutomationParameters"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSSMMaintenanceWindowTask_MaintenanceWindowAutomationParameters) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
