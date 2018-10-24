package cloudformation

// AWSAutoScalingLaunchConfiguration_BlockDeviceMapping AWS CloudFormation Resource (AWS::AutoScaling::LaunchConfiguration.BlockDeviceMapping)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-as-launchconfig-blockdev-mapping.html
type AWSAutoScalingLaunchConfiguration_BlockDeviceMapping struct {

	// DeviceName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-as-launchconfig-blockdev-mapping.html#cfn-as-launchconfig-blockdev-mapping-devicename
	DeviceName string `json:"DeviceName,omitempty"`

	// Ebs AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-as-launchconfig-blockdev-mapping.html#cfn-as-launchconfig-blockdev-mapping-ebs
	Ebs *AWSAutoScalingLaunchConfiguration_BlockDevice `json:"Ebs,omitempty"`

	// NoDevice AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-as-launchconfig-blockdev-mapping.html#cfn-as-launchconfig-blockdev-mapping-nodevice
	NoDevice bool `json:"NoDevice,omitempty"`

	// VirtualName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-as-launchconfig-blockdev-mapping.html#cfn-as-launchconfig-blockdev-mapping-virtualname
	VirtualName string `json:"VirtualName,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSAutoScalingLaunchConfiguration_BlockDeviceMapping) AWSCloudFormationType() string {
	return "AWS::AutoScaling::LaunchConfiguration.BlockDeviceMapping"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSAutoScalingLaunchConfiguration_BlockDeviceMapping) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
