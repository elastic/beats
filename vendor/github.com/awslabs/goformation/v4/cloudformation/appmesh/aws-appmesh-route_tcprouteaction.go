package appmesh

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Route_TcpRouteAction AWS CloudFormation Resource (AWS::AppMesh::Route.TcpRouteAction)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-route-tcprouteaction.html
type Route_TcpRouteAction struct {

	// WeightedTargets AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-route-tcprouteaction.html#cfn-appmesh-route-tcprouteaction-weightedtargets
	WeightedTargets []Route_WeightedTarget `json:"WeightedTargets,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Route_TcpRouteAction) AWSCloudFormationType() string {
	return "AWS::AppMesh::Route.TcpRouteAction"
}
