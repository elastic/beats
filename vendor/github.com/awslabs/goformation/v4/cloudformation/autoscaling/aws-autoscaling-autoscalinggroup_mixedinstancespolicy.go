package autoscaling

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// AutoScalingGroup_MixedInstancesPolicy AWS CloudFormation Resource (AWS::AutoScaling::AutoScalingGroup.MixedInstancesPolicy)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-as-group-mixedinstancespolicy.html
type AutoScalingGroup_MixedInstancesPolicy struct {

	// InstancesDistribution AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-as-group-mixedinstancespolicy.html#cfn-as-mixedinstancespolicy-instancesdistribution
	InstancesDistribution *AutoScalingGroup_InstancesDistribution `json:"InstancesDistribution,omitempty"`

	// LaunchTemplate AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-as-group-mixedinstancespolicy.html#cfn-as-mixedinstancespolicy-launchtemplate
	LaunchTemplate *AutoScalingGroup_LaunchTemplate `json:"LaunchTemplate,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AutoScalingGroup_MixedInstancesPolicy) AWSCloudFormationType() string {
	return "AWS::AutoScaling::AutoScalingGroup.MixedInstancesPolicy"
}
