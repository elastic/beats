package medialive

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Channel_InputAttachment AWS CloudFormation Resource (AWS::MediaLive::Channel.InputAttachment)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-inputattachment.html
type Channel_InputAttachment struct {

	// InputAttachmentName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-inputattachment.html#cfn-medialive-channel-inputattachment-inputattachmentname
	InputAttachmentName string `json:"InputAttachmentName,omitempty"`

	// InputId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-inputattachment.html#cfn-medialive-channel-inputattachment-inputid
	InputId string `json:"InputId,omitempty"`

	// InputSettings AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-inputattachment.html#cfn-medialive-channel-inputattachment-inputsettings
	InputSettings *Channel_InputSettings `json:"InputSettings,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Channel_InputAttachment) AWSCloudFormationType() string {
	return "AWS::MediaLive::Channel.InputAttachment"
}
