package ecs

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// TaskSet_Scale AWS CloudFormation Resource (AWS::ECS::TaskSet.Scale)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-taskset-scale.html
type TaskSet_Scale struct {

	// Unit AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-taskset-scale.html#cfn-ecs-taskset-scale-unit
	Unit string `json:"Unit,omitempty"`

	// Value AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-taskset-scale.html#cfn-ecs-taskset-scale-value
	Value float64 `json:"Value,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *TaskSet_Scale) AWSCloudFormationType() string {
	return "AWS::ECS::TaskSet.Scale"
}
