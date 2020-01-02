package medialive

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Channel_AudioSelectorSettings AWS CloudFormation Resource (AWS::MediaLive::Channel.AudioSelectorSettings)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-audioselectorsettings.html
type Channel_AudioSelectorSettings struct {

	// AudioLanguageSelection AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-audioselectorsettings.html#cfn-medialive-channel-audioselectorsettings-audiolanguageselection
	AudioLanguageSelection *Channel_AudioLanguageSelection `json:"AudioLanguageSelection,omitempty"`

	// AudioPidSelection AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-audioselectorsettings.html#cfn-medialive-channel-audioselectorsettings-audiopidselection
	AudioPidSelection *Channel_AudioPidSelection `json:"AudioPidSelection,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Channel_AudioSelectorSettings) AWSCloudFormationType() string {
	return "AWS::MediaLive::Channel.AudioSelectorSettings"
}
