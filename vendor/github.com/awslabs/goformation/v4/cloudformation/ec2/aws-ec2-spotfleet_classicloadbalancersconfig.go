package ec2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// SpotFleet_ClassicLoadBalancersConfig AWS CloudFormation Resource (AWS::EC2::SpotFleet.ClassicLoadBalancersConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-spotfleet-classicloadbalancersconfig.html
type SpotFleet_ClassicLoadBalancersConfig struct {

	// ClassicLoadBalancers AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-spotfleet-classicloadbalancersconfig.html#cfn-ec2-spotfleet-classicloadbalancersconfig-classicloadbalancers
	ClassicLoadBalancers []SpotFleet_ClassicLoadBalancer `json:"ClassicLoadBalancers,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *SpotFleet_ClassicLoadBalancersConfig) AWSCloudFormationType() string {
	return "AWS::EC2::SpotFleet.ClassicLoadBalancersConfig"
}
