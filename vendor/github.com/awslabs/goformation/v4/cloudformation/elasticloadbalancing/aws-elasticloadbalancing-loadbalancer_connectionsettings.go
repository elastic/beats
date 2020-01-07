package elasticloadbalancing

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// LoadBalancer_ConnectionSettings AWS CloudFormation Resource (AWS::ElasticLoadBalancing::LoadBalancer.ConnectionSettings)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb-connectionsettings.html
type LoadBalancer_ConnectionSettings struct {

	// IdleTimeout AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb-connectionsettings.html#cfn-elb-connectionsettings-idletimeout
	IdleTimeout int `json:"IdleTimeout"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *LoadBalancer_ConnectionSettings) AWSCloudFormationType() string {
	return "AWS::ElasticLoadBalancing::LoadBalancer.ConnectionSettings"
}
