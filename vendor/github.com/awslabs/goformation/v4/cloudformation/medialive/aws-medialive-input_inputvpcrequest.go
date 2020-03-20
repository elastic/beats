package medialive

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Input_InputVpcRequest AWS CloudFormation Resource (AWS::MediaLive::Input.InputVpcRequest)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-input-inputvpcrequest.html
type Input_InputVpcRequest struct {

	// SecurityGroupIds AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-input-inputvpcrequest.html#cfn-medialive-input-inputvpcrequest-securitygroupids
	SecurityGroupIds []string `json:"SecurityGroupIds,omitempty"`

	// SubnetIds AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-input-inputvpcrequest.html#cfn-medialive-input-inputvpcrequest-subnetids
	SubnetIds []string `json:"SubnetIds,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Input_InputVpcRequest) AWSCloudFormationType() string {
	return "AWS::MediaLive::Input.InputVpcRequest"
}
