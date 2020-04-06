package medialive

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Channel_MediaPackageOutputDestinationSettings AWS CloudFormation Resource (AWS::MediaLive::Channel.MediaPackageOutputDestinationSettings)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-mediapackageoutputdestinationsettings.html
type Channel_MediaPackageOutputDestinationSettings struct {

	// ChannelId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-mediapackageoutputdestinationsettings.html#cfn-medialive-channel-mediapackageoutputdestinationsettings-channelid
	ChannelId string `json:"ChannelId,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Channel_MediaPackageOutputDestinationSettings) AWSCloudFormationType() string {
	return "AWS::MediaLive::Channel.MediaPackageOutputDestinationSettings"
}
