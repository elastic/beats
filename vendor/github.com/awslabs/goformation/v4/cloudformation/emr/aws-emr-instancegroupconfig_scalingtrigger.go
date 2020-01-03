package emr

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// InstanceGroupConfig_ScalingTrigger AWS CloudFormation Resource (AWS::EMR::InstanceGroupConfig.ScalingTrigger)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-instancegroupconfig-scalingtrigger.html
type InstanceGroupConfig_ScalingTrigger struct {

	// CloudWatchAlarmDefinition AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-instancegroupconfig-scalingtrigger.html#cfn-elasticmapreduce-instancegroupconfig-scalingtrigger-cloudwatchalarmdefinition
	CloudWatchAlarmDefinition *InstanceGroupConfig_CloudWatchAlarmDefinition `json:"CloudWatchAlarmDefinition,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *InstanceGroupConfig_ScalingTrigger) AWSCloudFormationType() string {
	return "AWS::EMR::InstanceGroupConfig.ScalingTrigger"
}
