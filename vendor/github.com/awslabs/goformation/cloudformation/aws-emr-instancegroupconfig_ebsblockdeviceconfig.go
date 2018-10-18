package cloudformation

// AWSEMRInstanceGroupConfig_EbsBlockDeviceConfig AWS CloudFormation Resource (AWS::EMR::InstanceGroupConfig.EbsBlockDeviceConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-emr-ebsconfiguration-ebsblockdeviceconfig.html
type AWSEMRInstanceGroupConfig_EbsBlockDeviceConfig struct {

	// VolumeSpecification AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-emr-ebsconfiguration-ebsblockdeviceconfig.html#cfn-emr-ebsconfiguration-ebsblockdeviceconfig-volumespecification
	VolumeSpecification *AWSEMRInstanceGroupConfig_VolumeSpecification `json:"VolumeSpecification,omitempty"`

	// VolumesPerInstance AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-emr-ebsconfiguration-ebsblockdeviceconfig.html#cfn-emr-ebsconfiguration-ebsblockdeviceconfig-volumesperinstance
	VolumesPerInstance int `json:"VolumesPerInstance,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEMRInstanceGroupConfig_EbsBlockDeviceConfig) AWSCloudFormationType() string {
	return "AWS::EMR::InstanceGroupConfig.EbsBlockDeviceConfig"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEMRInstanceGroupConfig_EbsBlockDeviceConfig) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
