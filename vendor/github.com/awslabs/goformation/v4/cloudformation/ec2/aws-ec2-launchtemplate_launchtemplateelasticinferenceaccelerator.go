package ec2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// LaunchTemplate_LaunchTemplateElasticInferenceAccelerator AWS CloudFormation Resource (AWS::EC2::LaunchTemplate.LaunchTemplateElasticInferenceAccelerator)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-launchtemplate-launchtemplateelasticinferenceaccelerator.html
type LaunchTemplate_LaunchTemplateElasticInferenceAccelerator struct {

	// Type AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-launchtemplate-launchtemplateelasticinferenceaccelerator.html#cfn-ec2-launchtemplate-launchtemplateelasticinferenceaccelerator-type
	Type string `json:"Type,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *LaunchTemplate_LaunchTemplateElasticInferenceAccelerator) AWSCloudFormationType() string {
	return "AWS::EC2::LaunchTemplate.LaunchTemplateElasticInferenceAccelerator"
}
