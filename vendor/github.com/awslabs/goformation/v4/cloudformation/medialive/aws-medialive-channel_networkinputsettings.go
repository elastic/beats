package medialive

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Channel_NetworkInputSettings AWS CloudFormation Resource (AWS::MediaLive::Channel.NetworkInputSettings)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-networkinputsettings.html
type Channel_NetworkInputSettings struct {

	// HlsInputSettings AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-networkinputsettings.html#cfn-medialive-channel-networkinputsettings-hlsinputsettings
	HlsInputSettings *Channel_HlsInputSettings `json:"HlsInputSettings,omitempty"`

	// ServerValidation AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-networkinputsettings.html#cfn-medialive-channel-networkinputsettings-servervalidation
	ServerValidation string `json:"ServerValidation,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Channel_NetworkInputSettings) AWSCloudFormationType() string {
	return "AWS::MediaLive::Channel.NetworkInputSettings"
}
