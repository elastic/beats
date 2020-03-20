package medialive

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Channel_VideoSelectorPid AWS CloudFormation Resource (AWS::MediaLive::Channel.VideoSelectorPid)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-videoselectorpid.html
type Channel_VideoSelectorPid struct {

	// Pid AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-videoselectorpid.html#cfn-medialive-channel-videoselectorpid-pid
	Pid int `json:"Pid,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Channel_VideoSelectorPid) AWSCloudFormationType() string {
	return "AWS::MediaLive::Channel.VideoSelectorPid"
}
