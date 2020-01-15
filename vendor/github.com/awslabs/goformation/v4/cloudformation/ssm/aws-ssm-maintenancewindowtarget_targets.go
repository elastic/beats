package ssm

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// MaintenanceWindowTarget_Targets AWS CloudFormation Resource (AWS::SSM::MaintenanceWindowTarget.Targets)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtarget-targets.html
type MaintenanceWindowTarget_Targets struct {

	// Key AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtarget-targets.html#cfn-ssm-maintenancewindowtarget-targets-key
	Key string `json:"Key,omitempty"`

	// Values AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-maintenancewindowtarget-targets.html#cfn-ssm-maintenancewindowtarget-targets-values
	Values []string `json:"Values,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *MaintenanceWindowTarget_Targets) AWSCloudFormationType() string {
	return "AWS::SSM::MaintenanceWindowTarget.Targets"
}
