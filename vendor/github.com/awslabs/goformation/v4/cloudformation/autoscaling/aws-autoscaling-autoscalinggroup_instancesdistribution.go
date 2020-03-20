package autoscaling

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// AutoScalingGroup_InstancesDistribution AWS CloudFormation Resource (AWS::AutoScaling::AutoScalingGroup.InstancesDistribution)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-as-mixedinstancespolicy-instancesdistribution.html
type AutoScalingGroup_InstancesDistribution struct {

	// OnDemandAllocationStrategy AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-as-mixedinstancespolicy-instancesdistribution.html#cfn-autoscaling-autoscalinggroup-instancesdistribution-ondemandallocationstrategy
	OnDemandAllocationStrategy string `json:"OnDemandAllocationStrategy,omitempty"`

	// OnDemandBaseCapacity AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-as-mixedinstancespolicy-instancesdistribution.html#cfn-autoscaling-autoscalinggroup-instancesdistribution-ondemandbasecapacity
	OnDemandBaseCapacity int `json:"OnDemandBaseCapacity,omitempty"`

	// OnDemandPercentageAboveBaseCapacity AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-as-mixedinstancespolicy-instancesdistribution.html#cfn-autoscaling-autoscalinggroup-instancesdistribution-ondemandpercentageabovebasecapacity
	OnDemandPercentageAboveBaseCapacity int `json:"OnDemandPercentageAboveBaseCapacity,omitempty"`

	// SpotAllocationStrategy AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-as-mixedinstancespolicy-instancesdistribution.html#cfn-autoscaling-autoscalinggroup-instancesdistribution-spotallocationstrategy
	SpotAllocationStrategy string `json:"SpotAllocationStrategy,omitempty"`

	// SpotInstancePools AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-as-mixedinstancespolicy-instancesdistribution.html#cfn-autoscaling-autoscalinggroup-instancesdistribution-spotinstancepools
	SpotInstancePools int `json:"SpotInstancePools,omitempty"`

	// SpotMaxPrice AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-as-mixedinstancespolicy-instancesdistribution.html#cfn-autoscaling-autoscalinggroup-instancesdistribution-spotmaxprice
	SpotMaxPrice string `json:"SpotMaxPrice,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AutoScalingGroup_InstancesDistribution) AWSCloudFormationType() string {
	return "AWS::AutoScaling::AutoScalingGroup.InstancesDistribution"
}
