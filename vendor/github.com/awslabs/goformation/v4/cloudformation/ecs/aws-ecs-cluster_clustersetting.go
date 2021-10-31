package ecs

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Cluster_ClusterSetting AWS CloudFormation Resource (AWS::ECS::Cluster.ClusterSetting)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-cluster-clustersetting.html
type Cluster_ClusterSetting struct {

	// Name AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-cluster-clustersetting.html#cfn-ecs-cluster-clustersetting-name
	Name string `json:"Name,omitempty"`

	// Value AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ecs-cluster-clustersetting.html#cfn-ecs-cluster-clustersetting-value
	Value string `json:"Value,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Cluster_ClusterSetting) AWSCloudFormationType() string {
	return "AWS::ECS::Cluster.ClusterSetting"
}
