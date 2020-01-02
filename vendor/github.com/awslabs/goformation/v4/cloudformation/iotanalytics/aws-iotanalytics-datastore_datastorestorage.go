package iotanalytics

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Datastore_DatastoreStorage AWS CloudFormation Resource (AWS::IoTAnalytics::Datastore.DatastoreStorage)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-datastore-datastorestorage.html
type Datastore_DatastoreStorage struct {

	// CustomerManagedS3 AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-datastore-datastorestorage.html#cfn-iotanalytics-datastore-datastorestorage-customermanageds3
	CustomerManagedS3 *Datastore_CustomerManagedS3 `json:"CustomerManagedS3,omitempty"`

	// ServiceManagedS3 AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-datastore-datastorestorage.html#cfn-iotanalytics-datastore-datastorestorage-servicemanageds3
	ServiceManagedS3 *Datastore_ServiceManagedS3 `json:"ServiceManagedS3,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Datastore_DatastoreStorage) AWSCloudFormationType() string {
	return "AWS::IoTAnalytics::Datastore.DatastoreStorage"
}
