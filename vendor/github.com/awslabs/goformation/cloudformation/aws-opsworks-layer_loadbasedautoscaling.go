package cloudformation

// AWSOpsWorksLayer_LoadBasedAutoScaling AWS CloudFormation Resource (AWS::OpsWorks::Layer.LoadBasedAutoScaling)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-opsworks-layer-loadbasedautoscaling.html
type AWSOpsWorksLayer_LoadBasedAutoScaling struct {

	// DownScaling AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-opsworks-layer-loadbasedautoscaling.html#cfn-opsworks-layer-loadbasedautoscaling-downscaling
	DownScaling *AWSOpsWorksLayer_AutoScalingThresholds `json:"DownScaling,omitempty"`

	// Enable AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-opsworks-layer-loadbasedautoscaling.html#cfn-opsworks-layer-loadbasedautoscaling-enable
	Enable bool `json:"Enable,omitempty"`

	// UpScaling AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-opsworks-layer-loadbasedautoscaling.html#cfn-opsworks-layer-loadbasedautoscaling-upscaling
	UpScaling *AWSOpsWorksLayer_AutoScalingThresholds `json:"UpScaling,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSOpsWorksLayer_LoadBasedAutoScaling) AWSCloudFormationType() string {
	return "AWS::OpsWorks::Layer.LoadBasedAutoScaling"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSOpsWorksLayer_LoadBasedAutoScaling) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
