package cloudformation

// AWSAutoScalingScalingPolicy_StepAdjustment AWS CloudFormation Resource (AWS::AutoScaling::ScalingPolicy.StepAdjustment)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-autoscaling-scalingpolicy-stepadjustments.html
type AWSAutoScalingScalingPolicy_StepAdjustment struct {

	// MetricIntervalLowerBound AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-autoscaling-scalingpolicy-stepadjustments.html#cfn-autoscaling-scalingpolicy-stepadjustment-metricintervallowerbound
	MetricIntervalLowerBound float64 `json:"MetricIntervalLowerBound,omitempty"`

	// MetricIntervalUpperBound AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-autoscaling-scalingpolicy-stepadjustments.html#cfn-autoscaling-scalingpolicy-stepadjustment-metricintervalupperbound
	MetricIntervalUpperBound float64 `json:"MetricIntervalUpperBound,omitempty"`

	// ScalingAdjustment AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-autoscaling-scalingpolicy-stepadjustments.html#cfn-autoscaling-scalingpolicy-stepadjustment-scalingadjustment
	ScalingAdjustment int `json:"ScalingAdjustment,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSAutoScalingScalingPolicy_StepAdjustment) AWSCloudFormationType() string {
	return "AWS::AutoScaling::ScalingPolicy.StepAdjustment"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSAutoScalingScalingPolicy_StepAdjustment) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
