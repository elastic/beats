package medialive

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Channel_CaptionSelectorSettings AWS CloudFormation Resource (AWS::MediaLive::Channel.CaptionSelectorSettings)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-captionselectorsettings.html
type Channel_CaptionSelectorSettings struct {

	// AribSourceSettings AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-captionselectorsettings.html#cfn-medialive-channel-captionselectorsettings-aribsourcesettings
	AribSourceSettings *Channel_AribSourceSettings `json:"AribSourceSettings,omitempty"`

	// DvbSubSourceSettings AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-captionselectorsettings.html#cfn-medialive-channel-captionselectorsettings-dvbsubsourcesettings
	DvbSubSourceSettings *Channel_DvbSubSourceSettings `json:"DvbSubSourceSettings,omitempty"`

	// EmbeddedSourceSettings AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-captionselectorsettings.html#cfn-medialive-channel-captionselectorsettings-embeddedsourcesettings
	EmbeddedSourceSettings *Channel_EmbeddedSourceSettings `json:"EmbeddedSourceSettings,omitempty"`

	// Scte20SourceSettings AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-captionselectorsettings.html#cfn-medialive-channel-captionselectorsettings-scte20sourcesettings
	Scte20SourceSettings *Channel_Scte20SourceSettings `json:"Scte20SourceSettings,omitempty"`

	// Scte27SourceSettings AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-captionselectorsettings.html#cfn-medialive-channel-captionselectorsettings-scte27sourcesettings
	Scte27SourceSettings *Channel_Scte27SourceSettings `json:"Scte27SourceSettings,omitempty"`

	// TeletextSourceSettings AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-captionselectorsettings.html#cfn-medialive-channel-captionselectorsettings-teletextsourcesettings
	TeletextSourceSettings *Channel_TeletextSourceSettings `json:"TeletextSourceSettings,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Channel_CaptionSelectorSettings) AWSCloudFormationType() string {
	return "AWS::MediaLive::Channel.CaptionSelectorSettings"
}
