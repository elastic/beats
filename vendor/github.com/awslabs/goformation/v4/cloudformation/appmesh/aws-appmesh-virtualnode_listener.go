package appmesh

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// VirtualNode_Listener AWS CloudFormation Resource (AWS::AppMesh::VirtualNode.Listener)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-virtualnode-listener.html
type VirtualNode_Listener struct {

	// HealthCheck AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-virtualnode-listener.html#cfn-appmesh-virtualnode-listener-healthcheck
	HealthCheck *VirtualNode_HealthCheck `json:"HealthCheck,omitempty"`

	// PortMapping AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-virtualnode-listener.html#cfn-appmesh-virtualnode-listener-portmapping
	PortMapping *VirtualNode_PortMapping `json:"PortMapping,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *VirtualNode_Listener) AWSCloudFormationType() string {
	return "AWS::AppMesh::VirtualNode.Listener"
}
