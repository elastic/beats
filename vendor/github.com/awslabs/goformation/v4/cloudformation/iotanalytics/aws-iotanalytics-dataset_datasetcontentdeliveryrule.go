package iotanalytics

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Dataset_DatasetContentDeliveryRule AWS CloudFormation Resource (AWS::IoTAnalytics::Dataset.DatasetContentDeliveryRule)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-dataset-datasetcontentdeliveryrule.html
type Dataset_DatasetContentDeliveryRule struct {

	// Destination AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-dataset-datasetcontentdeliveryrule.html#cfn-iotanalytics-dataset-datasetcontentdeliveryrule-destination
	Destination *Dataset_DatasetContentDeliveryRuleDestination `json:"Destination,omitempty"`

	// EntryName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-dataset-datasetcontentdeliveryrule.html#cfn-iotanalytics-dataset-datasetcontentdeliveryrule-entryname
	EntryName string `json:"EntryName,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Dataset_DatasetContentDeliveryRule) AWSCloudFormationType() string {
	return "AWS::IoTAnalytics::Dataset.DatasetContentDeliveryRule"
}
