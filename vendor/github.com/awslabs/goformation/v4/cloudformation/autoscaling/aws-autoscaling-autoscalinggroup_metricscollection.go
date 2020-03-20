package autoscaling

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// AutoScalingGroup_MetricsCollection AWS CloudFormation Resource (AWS::AutoScaling::AutoScalingGroup.MetricsCollection)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-as-metricscollection.html
type AutoScalingGroup_MetricsCollection struct {

	// Granularity AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-as-metricscollection.html#cfn-as-metricscollection-granularity
	Granularity string `json:"Granularity,omitempty"`

	// Metrics AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-as-metricscollection.html#cfn-as-metricscollection-metrics
	Metrics []string `json:"Metrics,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AutoScalingGroup_MetricsCollection) AWSCloudFormationType() string {
	return "AWS::AutoScaling::AutoScalingGroup.MetricsCollection"
}
