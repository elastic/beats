package appmesh

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Mesh_EgressFilter AWS CloudFormation Resource (AWS::AppMesh::Mesh.EgressFilter)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-mesh-egressfilter.html
type Mesh_EgressFilter struct {

	// Type AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-mesh-egressfilter.html#cfn-appmesh-mesh-egressfilter-type
	Type string `json:"Type,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Mesh_EgressFilter) AWSCloudFormationType() string {
	return "AWS::AppMesh::Mesh.EgressFilter"
}
