package iotanalytics

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Dataset_Filter AWS CloudFormation Resource (AWS::IoTAnalytics::Dataset.Filter)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-dataset-filter.html
type Dataset_Filter struct {

	// DeltaTime AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-dataset-filter.html#cfn-iotanalytics-dataset-filter-deltatime
	DeltaTime *Dataset_DeltaTime `json:"DeltaTime,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Dataset_Filter) AWSCloudFormationType() string {
	return "AWS::IoTAnalytics::Dataset.Filter"
}
