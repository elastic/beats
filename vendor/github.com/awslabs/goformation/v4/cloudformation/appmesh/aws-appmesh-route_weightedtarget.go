package appmesh

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Route_WeightedTarget AWS CloudFormation Resource (AWS::AppMesh::Route.WeightedTarget)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-route-weightedtarget.html
type Route_WeightedTarget struct {

	// VirtualNode AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-route-weightedtarget.html#cfn-appmesh-route-weightedtarget-virtualnode
	VirtualNode string `json:"VirtualNode,omitempty"`

	// Weight AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-route-weightedtarget.html#cfn-appmesh-route-weightedtarget-weight
	Weight int `json:"Weight"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Route_WeightedTarget) AWSCloudFormationType() string {
	return "AWS::AppMesh::Route.WeightedTarget"
}
