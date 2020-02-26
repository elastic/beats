package appmesh

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// VirtualNode_FileAccessLog AWS CloudFormation Resource (AWS::AppMesh::VirtualNode.FileAccessLog)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-virtualnode-fileaccesslog.html
type VirtualNode_FileAccessLog struct {

	// Path AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-virtualnode-fileaccesslog.html#cfn-appmesh-virtualnode-fileaccesslog-path
	Path string `json:"Path,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *VirtualNode_FileAccessLog) AWSCloudFormationType() string {
	return "AWS::AppMesh::VirtualNode.FileAccessLog"
}
