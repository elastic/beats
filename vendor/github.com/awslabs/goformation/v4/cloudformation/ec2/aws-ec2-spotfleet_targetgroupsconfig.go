package ec2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// SpotFleet_TargetGroupsConfig AWS CloudFormation Resource (AWS::EC2::SpotFleet.TargetGroupsConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-spotfleet-targetgroupsconfig.html
type SpotFleet_TargetGroupsConfig struct {

	// TargetGroups AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-spotfleet-targetgroupsconfig.html#cfn-ec2-spotfleet-targetgroupsconfig-targetgroups
	TargetGroups []SpotFleet_TargetGroup `json:"TargetGroups,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *SpotFleet_TargetGroupsConfig) AWSCloudFormationType() string {
	return "AWS::EC2::SpotFleet.TargetGroupsConfig"
}
