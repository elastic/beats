package cloudformation

// AWSEMRCluster_ScalingAction AWS CloudFormation Resource (AWS::EMR::Cluster.ScalingAction)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-scalingaction.html
type AWSEMRCluster_ScalingAction struct {

	// Market AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-scalingaction.html#cfn-elasticmapreduce-cluster-scalingaction-market
	Market string `json:"Market,omitempty"`

	// SimpleScalingPolicyConfiguration AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-scalingaction.html#cfn-elasticmapreduce-cluster-scalingaction-simplescalingpolicyconfiguration
	SimpleScalingPolicyConfiguration *AWSEMRCluster_SimpleScalingPolicyConfiguration `json:"SimpleScalingPolicyConfiguration,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEMRCluster_ScalingAction) AWSCloudFormationType() string {
	return "AWS::EMR::Cluster.ScalingAction"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEMRCluster_ScalingAction) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
