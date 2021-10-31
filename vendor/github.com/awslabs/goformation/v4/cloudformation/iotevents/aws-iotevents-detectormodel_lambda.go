package iotevents

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// DetectorModel_Lambda AWS CloudFormation Resource (AWS::IoTEvents::DetectorModel.Lambda)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotevents-detectormodel-lambda.html
type DetectorModel_Lambda struct {

	// FunctionArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iotevents-detectormodel-lambda.html#cfn-iotevents-detectormodel-lambda-functionarn
	FunctionArn string `json:"FunctionArn,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *DetectorModel_Lambda) AWSCloudFormationType() string {
	return "AWS::IoTEvents::DetectorModel.Lambda"
}
