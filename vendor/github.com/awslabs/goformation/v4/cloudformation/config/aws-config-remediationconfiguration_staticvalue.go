package config

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// RemediationConfiguration_StaticValue AWS CloudFormation Resource (AWS::Config::RemediationConfiguration.StaticValue)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-config-remediationconfiguration-staticvalue.html
type RemediationConfiguration_StaticValue struct {

	// Values AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-config-remediationconfiguration-staticvalue.html#cfn-config-remediationconfiguration-staticvalue-values
	Values []string `json:"Values,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *RemediationConfiguration_StaticValue) AWSCloudFormationType() string {
	return "AWS::Config::RemediationConfiguration.StaticValue"
}
