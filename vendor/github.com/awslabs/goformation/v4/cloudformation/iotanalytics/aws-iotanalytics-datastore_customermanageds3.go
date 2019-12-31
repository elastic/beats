package iotanalytics

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Datastore_CustomerManagedS3 AWS CloudFormation Resource (AWS::IoTAnalytics::Datastore.CustomerManagedS3)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-datastore-customermanageds3.html
type Datastore_CustomerManagedS3 struct {

	// Bucket AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-datastore-customermanageds3.html#cfn-iotanalytics-datastore-customermanageds3-bucket
	Bucket string `json:"Bucket,omitempty"`

	// KeyPrefix AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-datastore-customermanageds3.html#cfn-iotanalytics-datastore-customermanageds3-keyprefix
	KeyPrefix string `json:"KeyPrefix,omitempty"`

	// RoleArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-datastore-customermanageds3.html#cfn-iotanalytics-datastore-customermanageds3-rolearn
	RoleArn string `json:"RoleArn,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Datastore_CustomerManagedS3) AWSCloudFormationType() string {
	return "AWS::IoTAnalytics::Datastore.CustomerManagedS3"
}
