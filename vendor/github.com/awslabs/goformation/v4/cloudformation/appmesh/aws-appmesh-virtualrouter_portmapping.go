package appmesh

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// VirtualRouter_PortMapping AWS CloudFormation Resource (AWS::AppMesh::VirtualRouter.PortMapping)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-virtualrouter-portmapping.html
type VirtualRouter_PortMapping struct {

	// Port AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-virtualrouter-portmapping.html#cfn-appmesh-virtualrouter-portmapping-port
	Port int `json:"Port"`

	// Protocol AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-virtualrouter-portmapping.html#cfn-appmesh-virtualrouter-portmapping-protocol
	Protocol string `json:"Protocol,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *VirtualRouter_PortMapping) AWSCloudFormationType() string {
	return "AWS::AppMesh::VirtualRouter.PortMapping"
}
