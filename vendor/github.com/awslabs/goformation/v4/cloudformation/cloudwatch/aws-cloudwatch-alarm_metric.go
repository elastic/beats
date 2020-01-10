package cloudwatch

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Alarm_Metric AWS CloudFormation Resource (AWS::CloudWatch::Alarm.Metric)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudwatch-alarm-metric.html
type Alarm_Metric struct {

	// Dimensions AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudwatch-alarm-metric.html#cfn-cloudwatch-alarm-metric-dimensions
	Dimensions []Alarm_Dimension `json:"Dimensions,omitempty"`

	// MetricName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudwatch-alarm-metric.html#cfn-cloudwatch-alarm-metric-metricname
	MetricName string `json:"MetricName,omitempty"`

	// Namespace AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudwatch-alarm-metric.html#cfn-cloudwatch-alarm-metric-namespace
	Namespace string `json:"Namespace,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Alarm_Metric) AWSCloudFormationType() string {
	return "AWS::CloudWatch::Alarm.Metric"
}
