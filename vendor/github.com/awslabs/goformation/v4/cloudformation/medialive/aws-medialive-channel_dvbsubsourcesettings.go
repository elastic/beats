package medialive

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Channel_DvbSubSourceSettings AWS CloudFormation Resource (AWS::MediaLive::Channel.DvbSubSourceSettings)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-dvbsubsourcesettings.html
type Channel_DvbSubSourceSettings struct {

	// Pid AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-channel-dvbsubsourcesettings.html#cfn-medialive-channel-dvbsubsourcesettings-pid
	Pid int `json:"Pid,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Channel_DvbSubSourceSettings) AWSCloudFormationType() string {
	return "AWS::MediaLive::Channel.DvbSubSourceSettings"
}
