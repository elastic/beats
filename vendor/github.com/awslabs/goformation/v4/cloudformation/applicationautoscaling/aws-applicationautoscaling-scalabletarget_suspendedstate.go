package applicationautoscaling

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ScalableTarget_SuspendedState AWS CloudFormation Resource (AWS::ApplicationAutoScaling::ScalableTarget.SuspendedState)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-applicationautoscaling-scalabletarget-suspendedstate.html
type ScalableTarget_SuspendedState struct {

	// DynamicScalingInSuspended AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-applicationautoscaling-scalabletarget-suspendedstate.html#cfn-applicationautoscaling-scalabletarget-suspendedstate-dynamicscalinginsuspended
	DynamicScalingInSuspended bool `json:"DynamicScalingInSuspended,omitempty"`

	// DynamicScalingOutSuspended AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-applicationautoscaling-scalabletarget-suspendedstate.html#cfn-applicationautoscaling-scalabletarget-suspendedstate-dynamicscalingoutsuspended
	DynamicScalingOutSuspended bool `json:"DynamicScalingOutSuspended,omitempty"`

	// ScheduledScalingSuspended AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-applicationautoscaling-scalabletarget-suspendedstate.html#cfn-applicationautoscaling-scalabletarget-suspendedstate-scheduledscalingsuspended
	ScheduledScalingSuspended bool `json:"ScheduledScalingSuspended,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ScalableTarget_SuspendedState) AWSCloudFormationType() string {
	return "AWS::ApplicationAutoScaling::ScalableTarget.SuspendedState"
}
