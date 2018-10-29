package cloudformation

// AWSEMRInstanceGroupConfig_ScalingConstraints AWS CloudFormation Resource (AWS::EMR::InstanceGroupConfig.ScalingConstraints)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-instancegroupconfig-scalingconstraints.html
type AWSEMRInstanceGroupConfig_ScalingConstraints struct {

	// MaxCapacity AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-instancegroupconfig-scalingconstraints.html#cfn-elasticmapreduce-instancegroupconfig-scalingconstraints-maxcapacity
	MaxCapacity int `json:"MaxCapacity,omitempty"`

	// MinCapacity AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-instancegroupconfig-scalingconstraints.html#cfn-elasticmapreduce-instancegroupconfig-scalingconstraints-mincapacity
	MinCapacity int `json:"MinCapacity,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEMRInstanceGroupConfig_ScalingConstraints) AWSCloudFormationType() string {
	return "AWS::EMR::InstanceGroupConfig.ScalingConstraints"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEMRInstanceGroupConfig_ScalingConstraints) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
