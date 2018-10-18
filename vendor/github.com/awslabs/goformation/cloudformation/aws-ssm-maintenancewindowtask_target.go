package cloudformation

// AWSSSMMaintenanceWindowTask_Target AWS CloudFormation Resource (AWS::SSM::MaintenanceWindowTask.Target)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-target.html
type AWSSSMMaintenanceWindowTask_Target struct {

	// Key AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-target.html#cfn-ssm-maintenancewindowtask-target-key
	Key string `json:"Key,omitempty"`

	// Values AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtask-target.html#cfn-ssm-maintenancewindowtask-target-values
	Values []string `json:"Values,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSSMMaintenanceWindowTask_Target) AWSCloudFormationType() string {
	return "AWS::SSM::MaintenanceWindowTask.Target"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSSMMaintenanceWindowTask_Target) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
