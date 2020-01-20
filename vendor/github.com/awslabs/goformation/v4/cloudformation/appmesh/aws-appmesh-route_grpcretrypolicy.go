package appmesh

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Route_GrpcRetryPolicy AWS CloudFormation Resource (AWS::AppMesh::Route.GrpcRetryPolicy)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-route-grpcretrypolicy.html
type Route_GrpcRetryPolicy struct {

	// GrpcRetryEvents AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-route-grpcretrypolicy.html#cfn-appmesh-route-grpcretrypolicy-grpcretryevents
	GrpcRetryEvents []string `json:"GrpcRetryEvents,omitempty"`

	// HttpRetryEvents AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-route-grpcretrypolicy.html#cfn-appmesh-route-grpcretrypolicy-httpretryevents
	HttpRetryEvents []string `json:"HttpRetryEvents,omitempty"`

	// MaxRetries AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-route-grpcretrypolicy.html#cfn-appmesh-route-grpcretrypolicy-maxretries
	MaxRetries int `json:"MaxRetries"`

	// PerRetryTimeout AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-route-grpcretrypolicy.html#cfn-appmesh-route-grpcretrypolicy-perretrytimeout
	PerRetryTimeout *Route_Duration `json:"PerRetryTimeout,omitempty"`

	// TcpRetryEvents AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appmesh-route-grpcretrypolicy.html#cfn-appmesh-route-grpcretrypolicy-tcpretryevents
	TcpRetryEvents []string `json:"TcpRetryEvents,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Route_GrpcRetryPolicy) AWSCloudFormationType() string {
	return "AWS::AppMesh::Route.GrpcRetryPolicy"
}
