package config

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// RemediationConfiguration_ExecutionControls AWS CloudFormation Resource (AWS::Config::RemediationConfiguration.ExecutionControls)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-config-remediationconfiguration-executioncontrols.html
type RemediationConfiguration_ExecutionControls struct {

	// SsmControls AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-config-remediationconfiguration-executioncontrols.html#cfn-config-remediationconfiguration-executioncontrols-ssmcontrols
	SsmControls *RemediationConfiguration_SsmControls `json:"SsmControls,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *RemediationConfiguration_ExecutionControls) AWSCloudFormationType() string {
	return "AWS::Config::RemediationConfiguration.ExecutionControls"
}
