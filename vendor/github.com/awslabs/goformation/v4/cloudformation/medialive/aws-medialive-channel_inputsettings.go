package medialive

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Channel_InputSettings AWS CloudFormation Resource (AWS::MediaLive::Channel.InputSettings)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-inputsettings.html
type Channel_InputSettings struct {

	// AudioSelectors AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-inputsettings.html#cfn-medialive-channel-inputsettings-audioselectors
	AudioSelectors []Channel_AudioSelector `json:"AudioSelectors,omitempty"`

	// CaptionSelectors AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-inputsettings.html#cfn-medialive-channel-inputsettings-captionselectors
	CaptionSelectors []Channel_CaptionSelector `json:"CaptionSelectors,omitempty"`

	// DeblockFilter AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-inputsettings.html#cfn-medialive-channel-inputsettings-deblockfilter
	DeblockFilter string `json:"DeblockFilter,omitempty"`

	// DenoiseFilter AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-inputsettings.html#cfn-medialive-channel-inputsettings-denoisefilter
	DenoiseFilter string `json:"DenoiseFilter,omitempty"`

	// FilterStrength AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-inputsettings.html#cfn-medialive-channel-inputsettings-filterstrength
	FilterStrength int `json:"FilterStrength,omitempty"`

	// InputFilter AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-inputsettings.html#cfn-medialive-channel-inputsettings-inputfilter
	InputFilter string `json:"InputFilter,omitempty"`

	// NetworkInputSettings AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-inputsettings.html#cfn-medialive-channel-inputsettings-networkinputsettings
	NetworkInputSettings *Channel_NetworkInputSettings `json:"NetworkInputSettings,omitempty"`

	// SourceEndBehavior AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-inputsettings.html#cfn-medialive-channel-inputsettings-sourceendbehavior
	SourceEndBehavior string `json:"SourceEndBehavior,omitempty"`

	// VideoSelector AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-inputsettings.html#cfn-medialive-channel-inputsettings-videoselector
	VideoSelector *Channel_VideoSelector `json:"VideoSelector,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Channel_InputSettings) AWSCloudFormationType() string {
	return "AWS::MediaLive::Channel.InputSettings"
}
