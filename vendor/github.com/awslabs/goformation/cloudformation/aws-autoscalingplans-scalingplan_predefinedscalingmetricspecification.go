package cloudformation

// AWSAutoScalingPlansScalingPlan_PredefinedScalingMetricSpecification AWS CloudFormation Resource (AWS::AutoScalingPlans::ScalingPlan.PredefinedScalingMetricSpecification)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-autoscalingplans-scalingplan-predefinedscalingmetricspecification.html
type AWSAutoScalingPlansScalingPlan_PredefinedScalingMetricSpecification struct {

	// PredefinedScalingMetricType AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-autoscalingplans-scalingplan-predefinedscalingmetricspecification.html#cfn-autoscalingplans-scalingplan-predefinedscalingmetricspecification-predefinedscalingmetrictype
	PredefinedScalingMetricType string `json:"PredefinedScalingMetricType,omitempty"`

	// ResourceLabel AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-autoscalingplans-scalingplan-predefinedscalingmetricspecification.html#cfn-autoscalingplans-scalingplan-predefinedscalingmetricspecification-resourcelabel
	ResourceLabel string `json:"ResourceLabel,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSAutoScalingPlansScalingPlan_PredefinedScalingMetricSpecification) AWSCloudFormationType() string {
	return "AWS::AutoScalingPlans::ScalingPlan.PredefinedScalingMetricSpecification"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSAutoScalingPlansScalingPlan_PredefinedScalingMetricSpecification) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
