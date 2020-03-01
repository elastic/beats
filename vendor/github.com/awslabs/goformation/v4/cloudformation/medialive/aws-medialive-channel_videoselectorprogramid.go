package medialive

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Channel_VideoSelectorProgramId AWS CloudFormation Resource (AWS::MediaLive::Channel.VideoSelectorProgramId)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-videoselectorprogramid.html
type Channel_VideoSelectorProgramId struct {

	// ProgramId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-videoselectorprogramid.html#cfn-medialive-channel-videoselectorprogramid-programid
	ProgramId int `json:"ProgramId,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Channel_VideoSelectorProgramId) AWSCloudFormationType() string {
	return "AWS::MediaLive::Channel.VideoSelectorProgramId"
}
