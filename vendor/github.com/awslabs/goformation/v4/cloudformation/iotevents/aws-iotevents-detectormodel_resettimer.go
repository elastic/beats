package iotevents

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// DetectorModel_ResetTimer AWS CloudFormation Resource (AWS::IoTEvents::DetectorModel.ResetTimer)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotevents-detectormodel-resettimer.html
type DetectorModel_ResetTimer struct {

	// TimerName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotevents-detectormodel-resettimer.html#cfn-iotevents-detectormodel-resettimer-timername
	TimerName string `json:"TimerName,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *DetectorModel_ResetTimer) AWSCloudFormationType() string {
	return "AWS::IoTEvents::DetectorModel.ResetTimer"
}
