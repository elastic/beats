package medialive

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Input_InputSourceRequest AWS CloudFormation Resource (AWS::MediaLive::Input.InputSourceRequest)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-input-inputsourcerequest.html
type Input_InputSourceRequest struct {

	// PasswordParam AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-input-inputsourcerequest.html#cfn-medialive-input-inputsourcerequest-passwordparam
	PasswordParam string `json:"PasswordParam,omitempty"`

	// Url AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-input-inputsourcerequest.html#cfn-medialive-input-inputsourcerequest-url
	Url string `json:"Url,omitempty"`

	// Username AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-input-inputsourcerequest.html#cfn-medialive-input-inputsourcerequest-username
	Username string `json:"Username,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Input_InputSourceRequest) AWSCloudFormationType() string {
	return "AWS::MediaLive::Input.InputSourceRequest"
}
