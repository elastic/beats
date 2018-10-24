package cloudformation

// AWSEMRCluster_BootstrapActionConfig AWS CloudFormation Resource (AWS::EMR::Cluster.BootstrapActionConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-bootstrapactionconfig.html
type AWSEMRCluster_BootstrapActionConfig struct {

	// Name AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-bootstrapactionconfig.html#cfn-elasticmapreduce-cluster-bootstrapactionconfig-name
	Name string `json:"Name,omitempty"`

	// ScriptBootstrapAction AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-bootstrapactionconfig.html#cfn-elasticmapreduce-cluster-bootstrapactionconfig-scriptbootstrapaction
	ScriptBootstrapAction *AWSEMRCluster_ScriptBootstrapActionConfig `json:"ScriptBootstrapAction,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEMRCluster_BootstrapActionConfig) AWSCloudFormationType() string {
	return "AWS::EMR::Cluster.BootstrapActionConfig"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEMRCluster_BootstrapActionConfig) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
