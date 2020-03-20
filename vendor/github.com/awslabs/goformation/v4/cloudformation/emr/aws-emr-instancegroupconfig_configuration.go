package emr

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// InstanceGroupConfig_Configuration AWS CloudFormation Resource (AWS::EMR::InstanceGroupConfig.Configuration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-emr-cluster-configuration.html
type InstanceGroupConfig_Configuration struct {

	// Classification AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-emr-cluster-configuration.html#cfn-emr-cluster-configuration-classification
	Classification string `json:"Classification,omitempty"`

	// ConfigurationProperties AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-emr-cluster-configuration.html#cfn-emr-cluster-configuration-configurationproperties
	ConfigurationProperties map[string]string `json:"ConfigurationProperties,omitempty"`

	// Configurations AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-emr-cluster-configuration.html#cfn-emr-cluster-configuration-configurations
	Configurations []InstanceGroupConfig_Configuration `json:"Configurations,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *InstanceGroupConfig_Configuration) AWSCloudFormationType() string {
	return "AWS::EMR::InstanceGroupConfig.Configuration"
}
