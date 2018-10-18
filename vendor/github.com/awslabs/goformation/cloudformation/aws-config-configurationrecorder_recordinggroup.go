package cloudformation

// AWSConfigConfigurationRecorder_RecordingGroup AWS CloudFormation Resource (AWS::Config::ConfigurationRecorder.RecordingGroup)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-config-configurationrecorder-recordinggroup.html
type AWSConfigConfigurationRecorder_RecordingGroup struct {

	// AllSupported AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-config-configurationrecorder-recordinggroup.html#cfn-config-configurationrecorder-recordinggroup-allsupported
	AllSupported bool `json:"AllSupported,omitempty"`

	// IncludeGlobalResourceTypes AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-config-configurationrecorder-recordinggroup.html#cfn-config-configurationrecorder-recordinggroup-includeglobalresourcetypes
	IncludeGlobalResourceTypes bool `json:"IncludeGlobalResourceTypes,omitempty"`

	// ResourceTypes AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-config-configurationrecorder-recordinggroup.html#cfn-config-configurationrecorder-recordinggroup-resourcetypes
	ResourceTypes []string `json:"ResourceTypes,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSConfigConfigurationRecorder_RecordingGroup) AWSCloudFormationType() string {
	return "AWS::Config::ConfigurationRecorder.RecordingGroup"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSConfigConfigurationRecorder_RecordingGroup) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
