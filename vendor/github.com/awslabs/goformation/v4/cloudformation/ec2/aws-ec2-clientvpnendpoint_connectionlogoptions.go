package ec2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ClientVpnEndpoint_ConnectionLogOptions AWS CloudFormation Resource (AWS::EC2::ClientVpnEndpoint.ConnectionLogOptions)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-clientvpnendpoint-connectionlogoptions.html
type ClientVpnEndpoint_ConnectionLogOptions struct {

	// CloudwatchLogGroup AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-clientvpnendpoint-connectionlogoptions.html#cfn-ec2-clientvpnendpoint-connectionlogoptions-cloudwatchloggroup
	CloudwatchLogGroup string `json:"CloudwatchLogGroup,omitempty"`

	// CloudwatchLogStream AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-clientvpnendpoint-connectionlogoptions.html#cfn-ec2-clientvpnendpoint-connectionlogoptions-cloudwatchlogstream
	CloudwatchLogStream string `json:"CloudwatchLogStream,omitempty"`

	// Enabled AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-clientvpnendpoint-connectionlogoptions.html#cfn-ec2-clientvpnendpoint-connectionlogoptions-enabled
	Enabled bool `json:"Enabled"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ClientVpnEndpoint_ConnectionLogOptions) AWSCloudFormationType() string {
	return "AWS::EC2::ClientVpnEndpoint.ConnectionLogOptions"
}
