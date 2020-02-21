package emr

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// InstanceGroupConfig_AutoScalingPolicy AWS CloudFormation Resource (AWS::EMR::InstanceGroupConfig.AutoScalingPolicy)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-instancegroupconfig-autoscalingpolicy.html
type InstanceGroupConfig_AutoScalingPolicy struct {

	// Constraints AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-instancegroupconfig-autoscalingpolicy.html#cfn-elasticmapreduce-instancegroupconfig-autoscalingpolicy-constraints
	Constraints *InstanceGroupConfig_ScalingConstraints `json:"Constraints,omitempty"`

	// Rules AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-instancegroupconfig-autoscalingpolicy.html#cfn-elasticmapreduce-instancegroupconfig-autoscalingpolicy-rules
	Rules []InstanceGroupConfig_ScalingRule `json:"Rules,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *InstanceGroupConfig_AutoScalingPolicy) AWSCloudFormationType() string {
	return "AWS::EMR::InstanceGroupConfig.AutoScalingPolicy"
}
