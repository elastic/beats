package emr

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Cluster_BootstrapActionConfig AWS CloudFormation Resource (AWS::EMR::Cluster.BootstrapActionConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-bootstrapactionconfig.html
type Cluster_BootstrapActionConfig struct {

	// Name AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-bootstrapactionconfig.html#cfn-elasticmapreduce-cluster-bootstrapactionconfig-name
	Name string `json:"Name,omitempty"`

	// ScriptBootstrapAction AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-bootstrapactionconfig.html#cfn-elasticmapreduce-cluster-bootstrapactionconfig-scriptbootstrapaction
	ScriptBootstrapAction *Cluster_ScriptBootstrapActionConfig `json:"ScriptBootstrapAction,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Cluster_BootstrapActionConfig) AWSCloudFormationType() string {
	return "AWS::EMR::Cluster.BootstrapActionConfig"
}
