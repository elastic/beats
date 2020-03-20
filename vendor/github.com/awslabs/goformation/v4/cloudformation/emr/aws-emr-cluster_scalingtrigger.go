package emr

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Cluster_ScalingTrigger AWS CloudFormation Resource (AWS::EMR::Cluster.ScalingTrigger)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-scalingtrigger.html
type Cluster_ScalingTrigger struct {

	// CloudWatchAlarmDefinition AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-scalingtrigger.html#cfn-elasticmapreduce-cluster-scalingtrigger-cloudwatchalarmdefinition
	CloudWatchAlarmDefinition *Cluster_CloudWatchAlarmDefinition `json:"CloudWatchAlarmDefinition,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Cluster_ScalingTrigger) AWSCloudFormationType() string {
	return "AWS::EMR::Cluster.ScalingTrigger"
}
