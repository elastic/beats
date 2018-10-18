package cloudformation

// AWSOpsWorksLayer_AutoScalingThresholds AWS CloudFormation Resource (AWS::OpsWorks::Layer.AutoScalingThresholds)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-opsworks-layer-loadbasedautoscaling-autoscalingthresholds.html
type AWSOpsWorksLayer_AutoScalingThresholds struct {

	// CpuThreshold AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-opsworks-layer-loadbasedautoscaling-autoscalingthresholds.html#cfn-opsworks-layer-loadbasedautoscaling-autoscalingthresholds-cputhreshold
	CpuThreshold float64 `json:"CpuThreshold,omitempty"`

	// IgnoreMetricsTime AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-opsworks-layer-loadbasedautoscaling-autoscalingthresholds.html#cfn-opsworks-layer-loadbasedautoscaling-autoscalingthresholds-ignoremetricstime
	IgnoreMetricsTime int `json:"IgnoreMetricsTime,omitempty"`

	// InstanceCount AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-opsworks-layer-loadbasedautoscaling-autoscalingthresholds.html#cfn-opsworks-layer-loadbasedautoscaling-autoscalingthresholds-instancecount
	InstanceCount int `json:"InstanceCount,omitempty"`

	// LoadThreshold AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-opsworks-layer-loadbasedautoscaling-autoscalingthresholds.html#cfn-opsworks-layer-loadbasedautoscaling-autoscalingthresholds-loadthreshold
	LoadThreshold float64 `json:"LoadThreshold,omitempty"`

	// MemoryThreshold AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-opsworks-layer-loadbasedautoscaling-autoscalingthresholds.html#cfn-opsworks-layer-loadbasedautoscaling-autoscalingthresholds-memorythreshold
	MemoryThreshold float64 `json:"MemoryThreshold,omitempty"`

	// ThresholdsWaitTime AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-opsworks-layer-loadbasedautoscaling-autoscalingthresholds.html#cfn-opsworks-layer-loadbasedautoscaling-autoscalingthresholds-thresholdwaittime
	ThresholdsWaitTime int `json:"ThresholdsWaitTime,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSOpsWorksLayer_AutoScalingThresholds) AWSCloudFormationType() string {
	return "AWS::OpsWorks::Layer.AutoScalingThresholds"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSOpsWorksLayer_AutoScalingThresholds) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
