package config

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// RemediationConfiguration_SsmControls AWS CloudFormation Resource (AWS::Config::RemediationConfiguration.SsmControls)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-config-remediationconfiguration-ssmcontrols.html
type RemediationConfiguration_SsmControls struct {

	// ConcurrentExecutionRatePercentage AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-config-remediationconfiguration-ssmcontrols.html#cfn-config-remediationconfiguration-ssmcontrols-concurrentexecutionratepercentage
	ConcurrentExecutionRatePercentage int `json:"ConcurrentExecutionRatePercentage,omitempty"`

	// ErrorPercentage AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-config-remediationconfiguration-ssmcontrols.html#cfn-config-remediationconfiguration-ssmcontrols-errorpercentage
	ErrorPercentage int `json:"ErrorPercentage,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *RemediationConfiguration_SsmControls) AWSCloudFormationType() string {
	return "AWS::Config::RemediationConfiguration.SsmControls"
}
