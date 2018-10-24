package cloudformation

// AWSEMRInstanceGroupConfig_ScalingTrigger AWS CloudFormation Resource (AWS::EMR::InstanceGroupConfig.ScalingTrigger)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-instancegroupconfig-scalingtrigger.html
type AWSEMRInstanceGroupConfig_ScalingTrigger struct {

	// CloudWatchAlarmDefinition AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-instancegroupconfig-scalingtrigger.html#cfn-elasticmapreduce-instancegroupconfig-scalingtrigger-cloudwatchalarmdefinition
	CloudWatchAlarmDefinition *AWSEMRInstanceGroupConfig_CloudWatchAlarmDefinition `json:"CloudWatchAlarmDefinition,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEMRInstanceGroupConfig_ScalingTrigger) AWSCloudFormationType() string {
	return "AWS::EMR::InstanceGroupConfig.ScalingTrigger"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEMRInstanceGroupConfig_ScalingTrigger) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
