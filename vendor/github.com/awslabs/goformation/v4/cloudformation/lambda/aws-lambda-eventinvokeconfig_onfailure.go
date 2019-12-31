package lambda

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// EventInvokeConfig_OnFailure AWS CloudFormation Resource (AWS::Lambda::EventInvokeConfig.OnFailure)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-eventinvokeconfig-destinationconfig-onfailure.html
type EventInvokeConfig_OnFailure struct {

	// Destination AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-eventinvokeconfig-destinationconfig-onfailure.html#cfn-lambda-eventinvokeconfig-destinationconfig-onfailure-destination
	Destination string `json:"Destination,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *EventInvokeConfig_OnFailure) AWSCloudFormationType() string {
	return "AWS::Lambda::EventInvokeConfig.OnFailure"
}
