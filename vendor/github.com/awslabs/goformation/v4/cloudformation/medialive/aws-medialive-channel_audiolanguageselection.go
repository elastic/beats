package medialive

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Channel_AudioLanguageSelection AWS CloudFormation Resource (AWS::MediaLive::Channel.AudioLanguageSelection)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-audiolanguageselection.html
type Channel_AudioLanguageSelection struct {

	// LanguageCode AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-audiolanguageselection.html#cfn-medialive-channel-audiolanguageselection-languagecode
	LanguageCode string `json:"LanguageCode,omitempty"`

	// LanguageSelectionPolicy AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-audiolanguageselection.html#cfn-medialive-channel-audiolanguageselection-languageselectionpolicy
	LanguageSelectionPolicy string `json:"LanguageSelectionPolicy,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Channel_AudioLanguageSelection) AWSCloudFormationType() string {
	return "AWS::MediaLive::Channel.AudioLanguageSelection"
}
