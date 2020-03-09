package autoscalingplans

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ScalingPlan_ApplicationSource AWS CloudFormation Resource (AWS::AutoScalingPlans::ScalingPlan.ApplicationSource)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-autoscalingplans-scalingplan-applicationsource.html
type ScalingPlan_ApplicationSource struct {

	// CloudFormationStackARN AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-autoscalingplans-scalingplan-applicationsource.html#cfn-autoscalingplans-scalingplan-applicationsource-cloudformationstackarn
	CloudFormationStackARN string `json:"CloudFormationStackARN,omitempty"`

	// TagFilters AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-autoscalingplans-scalingplan-applicationsource.html#cfn-autoscalingplans-scalingplan-applicationsource-tagfilters
	TagFilters []ScalingPlan_TagFilter `json:"TagFilters,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ScalingPlan_ApplicationSource) AWSCloudFormationType() string {
	return "AWS::AutoScalingPlans::ScalingPlan.ApplicationSource"
}
