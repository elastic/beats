package appmesh

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// VirtualNode_AwsCloudMapInstanceAttribute AWS CloudFormation Resource (AWS::AppMesh::VirtualNode.AwsCloudMapInstanceAttribute)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-virtualnode-awscloudmapinstanceattribute.html
type VirtualNode_AwsCloudMapInstanceAttribute struct {

	// Key AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-virtualnode-awscloudmapinstanceattribute.html#cfn-appmesh-virtualnode-awscloudmapinstanceattribute-key
	Key string `json:"Key,omitempty"`

	// Value AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-virtualnode-awscloudmapinstanceattribute.html#cfn-appmesh-virtualnode-awscloudmapinstanceattribute-value
	Value string `json:"Value,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *VirtualNode_AwsCloudMapInstanceAttribute) AWSCloudFormationType() string {
	return "AWS::AppMesh::VirtualNode.AwsCloudMapInstanceAttribute"
}
