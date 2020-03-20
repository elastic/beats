package iotevents

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// DetectorModel_IotTopicPublish AWS CloudFormation Resource (AWS::IoTEvents::DetectorModel.IotTopicPublish)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotevents-detectormodel-iottopicpublish.html
type DetectorModel_IotTopicPublish struct {

	// MqttTopic AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotevents-detectormodel-iottopicpublish.html#cfn-iotevents-detectormodel-iottopicpublish-mqtttopic
	MqttTopic string `json:"MqttTopic,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *DetectorModel_IotTopicPublish) AWSCloudFormationType() string {
	return "AWS::IoTEvents::DetectorModel.IotTopicPublish"
}
