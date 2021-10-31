package ecs

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// TaskDefinition_KernelCapabilities AWS CloudFormation Resource (AWS::ECS::TaskDefinition.KernelCapabilities)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-taskdefinition-kernelcapabilities.html
type TaskDefinition_KernelCapabilities struct {

	// Add AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-taskdefinition-kernelcapabilities.html#cfn-ecs-taskdefinition-kernelcapabilities-add
	Add []string `json:"Add,omitempty"`

	// Drop AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-taskdefinition-kernelcapabilities.html#cfn-ecs-taskdefinition-kernelcapabilities-drop
	Drop []string `json:"Drop,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *TaskDefinition_KernelCapabilities) AWSCloudFormationType() string {
	return "AWS::ECS::TaskDefinition.KernelCapabilities"
}
