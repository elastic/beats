package pinpoint

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Campaign_MetricDimension AWS CloudFormation Resource (AWS::Pinpoint::Campaign.MetricDimension)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpoint-campaign-metricdimension.html
type Campaign_MetricDimension struct {

	// ComparisonOperator AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpoint-campaign-metricdimension.html#cfn-pinpoint-campaign-metricdimension-comparisonoperator
	ComparisonOperator string `json:"ComparisonOperator,omitempty"`

	// Value AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpoint-campaign-metricdimension.html#cfn-pinpoint-campaign-metricdimension-value
	Value float64 `json:"Value,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Campaign_MetricDimension) AWSCloudFormationType() string {
	return "AWS::Pinpoint::Campaign.MetricDimension"
}
