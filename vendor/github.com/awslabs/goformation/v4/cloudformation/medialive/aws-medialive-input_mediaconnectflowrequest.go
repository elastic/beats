package medialive

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Input_MediaConnectFlowRequest AWS CloudFormation Resource (AWS::MediaLive::Input.MediaConnectFlowRequest)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-input-mediaconnectflowrequest.html
type Input_MediaConnectFlowRequest struct {

	// FlowArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-input-mediaconnectflowrequest.html#cfn-medialive-input-mediaconnectflowrequest-flowarn
	FlowArn string `json:"FlowArn,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Input_MediaConnectFlowRequest) AWSCloudFormationType() string {
	return "AWS::MediaLive::Input.MediaConnectFlowRequest"
}
