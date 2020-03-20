package ecs

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// TaskSet_NetworkConfiguration AWS CloudFormation Resource (AWS::ECS::TaskSet.NetworkConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-taskset-networkconfiguration.html
type TaskSet_NetworkConfiguration struct {

	// AwsVpcConfiguration AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-taskset-networkconfiguration.html#cfn-ecs-taskset-networkconfiguration-awsvpcconfiguration
	AwsVpcConfiguration *TaskSet_AwsVpcConfiguration `json:"AwsVpcConfiguration,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *TaskSet_NetworkConfiguration) AWSCloudFormationType() string {
	return "AWS::ECS::TaskSet.NetworkConfiguration"
}
