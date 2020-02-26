package iotanalytics

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Datastore_ServiceManagedS3 AWS CloudFormation Resource (AWS::IoTAnalytics::Datastore.ServiceManagedS3)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-datastore-servicemanageds3.html
type Datastore_ServiceManagedS3 struct {

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Datastore_ServiceManagedS3) AWSCloudFormationType() string {
	return "AWS::IoTAnalytics::Datastore.ServiceManagedS3"
}
