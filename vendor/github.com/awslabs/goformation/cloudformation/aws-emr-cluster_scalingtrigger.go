package cloudformation

// AWSEMRCluster_ScalingTrigger AWS CloudFormation Resource (AWS::EMR::Cluster.ScalingTrigger)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-scalingtrigger.html
type AWSEMRCluster_ScalingTrigger struct {

	// CloudWatchAlarmDefinition AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-scalingtrigger.html#cfn-elasticmapreduce-cluster-scalingtrigger-cloudwatchalarmdefinition
	CloudWatchAlarmDefinition *AWSEMRCluster_CloudWatchAlarmDefinition `json:"CloudWatchAlarmDefinition,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEMRCluster_ScalingTrigger) AWSCloudFormationType() string {
	return "AWS::EMR::Cluster.ScalingTrigger"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEMRCluster_ScalingTrigger) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
