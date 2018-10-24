package cloudformation

// AWSAutoScalingPlansScalingPlan_ScalingInstruction AWS CloudFormation Resource (AWS::AutoScalingPlans::ScalingPlan.ScalingInstruction)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-autoscalingplans-scalingplan-scalinginstruction.html
type AWSAutoScalingPlansScalingPlan_ScalingInstruction struct {

	// MaxCapacity AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-autoscalingplans-scalingplan-scalinginstruction.html#cfn-autoscalingplans-scalingplan-scalinginstruction-maxcapacity
	MaxCapacity int `json:"MaxCapacity,omitempty"`

	// MinCapacity AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-autoscalingplans-scalingplan-scalinginstruction.html#cfn-autoscalingplans-scalingplan-scalinginstruction-mincapacity
	MinCapacity int `json:"MinCapacity,omitempty"`

	// ResourceId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-autoscalingplans-scalingplan-scalinginstruction.html#cfn-autoscalingplans-scalingplan-scalinginstruction-resourceid
	ResourceId string `json:"ResourceId,omitempty"`

	// ScalableDimension AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-autoscalingplans-scalingplan-scalinginstruction.html#cfn-autoscalingplans-scalingplan-scalinginstruction-scalabledimension
	ScalableDimension string `json:"ScalableDimension,omitempty"`

	// ServiceNamespace AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-autoscalingplans-scalingplan-scalinginstruction.html#cfn-autoscalingplans-scalingplan-scalinginstruction-servicenamespace
	ServiceNamespace string `json:"ServiceNamespace,omitempty"`

	// TargetTrackingConfigurations AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-autoscalingplans-scalingplan-scalinginstruction.html#cfn-autoscalingplans-scalingplan-scalinginstruction-targettrackingconfigurations
	TargetTrackingConfigurations []AWSAutoScalingPlansScalingPlan_TargetTrackingConfiguration `json:"TargetTrackingConfigurations,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSAutoScalingPlansScalingPlan_ScalingInstruction) AWSCloudFormationType() string {
	return "AWS::AutoScalingPlans::ScalingPlan.ScalingInstruction"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSAutoScalingPlansScalingPlan_ScalingInstruction) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
