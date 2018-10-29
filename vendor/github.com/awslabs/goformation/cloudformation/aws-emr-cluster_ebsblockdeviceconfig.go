package cloudformation

// AWSEMRCluster_EbsBlockDeviceConfig AWS CloudFormation Resource (AWS::EMR::Cluster.EbsBlockDeviceConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-ebsblockdeviceconfig.html
type AWSEMRCluster_EbsBlockDeviceConfig struct {

	// VolumeSpecification AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-ebsblockdeviceconfig.html#cfn-elasticmapreduce-cluster-ebsblockdeviceconfig-volumespecification
	VolumeSpecification *AWSEMRCluster_VolumeSpecification `json:"VolumeSpecification,omitempty"`

	// VolumesPerInstance AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-ebsblockdeviceconfig.html#cfn-elasticmapreduce-cluster-ebsblockdeviceconfig-volumesperinstance
	VolumesPerInstance int `json:"VolumesPerInstance,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEMRCluster_EbsBlockDeviceConfig) AWSCloudFormationType() string {
	return "AWS::EMR::Cluster.EbsBlockDeviceConfig"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEMRCluster_EbsBlockDeviceConfig) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
