package iotanalytics

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Dataset_OutputFileUriValue AWS CloudFormation Resource (AWS::IoTAnalytics::Dataset.OutputFileUriValue)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-dataset-variable-outputfileurivalue.html
type Dataset_OutputFileUriValue struct {

	// FileName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-dataset-variable-outputfileurivalue.html#cfn-iotanalytics-dataset-variable-outputfileurivalue-filename
	FileName string `json:"FileName,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Dataset_OutputFileUriValue) AWSCloudFormationType() string {
	return "AWS::IoTAnalytics::Dataset.OutputFileUriValue"
}
