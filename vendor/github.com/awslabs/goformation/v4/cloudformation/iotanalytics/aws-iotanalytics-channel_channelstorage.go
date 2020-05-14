package iotanalytics

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Channel_ChannelStorage AWS CloudFormation Resource (AWS::IoTAnalytics::Channel.ChannelStorage)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-channel-channelstorage.html
type Channel_ChannelStorage struct {

	// CustomerManagedS3 AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-channel-channelstorage.html#cfn-iotanalytics-channel-channelstorage-customermanageds3
	CustomerManagedS3 *Channel_CustomerManagedS3 `json:"CustomerManagedS3,omitempty"`

	// ServiceManagedS3 AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotanalytics-channel-channelstorage.html#cfn-iotanalytics-channel-channelstorage-servicemanageds3
	ServiceManagedS3 *Channel_ServiceManagedS3 `json:"ServiceManagedS3,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Channel_ChannelStorage) AWSCloudFormationType() string {
	return "AWS::IoTAnalytics::Channel.ChannelStorage"
}
