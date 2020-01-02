package ec2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// LaunchTemplate_CpuOptions AWS CloudFormation Resource (AWS::EC2::LaunchTemplate.CpuOptions)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-launchtemplate-launchtemplatedata-cpuoptions.html
type LaunchTemplate_CpuOptions struct {

	// CoreCount AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-launchtemplate-launchtemplatedata-cpuoptions.html#cfn-ec2-launchtemplate-launchtemplatedata-cpuoptions-corecount
	CoreCount int `json:"CoreCount,omitempty"`

	// ThreadsPerCore AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-launchtemplate-launchtemplatedata-cpuoptions.html#cfn-ec2-launchtemplate-launchtemplatedata-cpuoptions-threadspercore
	ThreadsPerCore int `json:"ThreadsPerCore,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *LaunchTemplate_CpuOptions) AWSCloudFormationType() string {
	return "AWS::EC2::LaunchTemplate.CpuOptions"
}
