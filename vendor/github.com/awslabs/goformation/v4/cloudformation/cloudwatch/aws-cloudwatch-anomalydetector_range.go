package cloudwatch

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// AnomalyDetector_Range AWS CloudFormation Resource (AWS::CloudWatch::AnomalyDetector.Range)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudwatch-anomalydetector-range.html
type AnomalyDetector_Range struct {

	// EndTime AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudwatch-anomalydetector-range.html#cfn-cloudwatch-anomalydetector-range-endtime
	EndTime string `json:"EndTime,omitempty"`

	// StartTime AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudwatch-anomalydetector-range.html#cfn-cloudwatch-anomalydetector-range-starttime
	StartTime string `json:"StartTime,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AnomalyDetector_Range) AWSCloudFormationType() string {
	return "AWS::CloudWatch::AnomalyDetector.Range"
}
