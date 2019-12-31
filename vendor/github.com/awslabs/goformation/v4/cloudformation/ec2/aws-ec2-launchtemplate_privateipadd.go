package ec2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// LaunchTemplate_PrivateIpAdd AWS CloudFormation Resource (AWS::EC2::LaunchTemplate.PrivateIpAdd)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-launchtemplate-privateipadd.html
type LaunchTemplate_PrivateIpAdd struct {

	// Primary AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-launchtemplate-privateipadd.html#cfn-ec2-launchtemplate-privateipadd-primary
	Primary bool `json:"Primary,omitempty"`

	// PrivateIpAddress AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-launchtemplate-privateipadd.html#cfn-ec2-launchtemplate-privateipadd-privateipaddress
	PrivateIpAddress string `json:"PrivateIpAddress,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *LaunchTemplate_PrivateIpAdd) AWSCloudFormationType() string {
	return "AWS::EC2::LaunchTemplate.PrivateIpAdd"
}
