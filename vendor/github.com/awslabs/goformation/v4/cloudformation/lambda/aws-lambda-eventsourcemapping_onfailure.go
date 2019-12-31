package lambda

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// EventSourceMapping_OnFailure AWS CloudFormation Resource (AWS::Lambda::EventSourceMapping.OnFailure)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-eventsourcemapping-onfailure.html
type EventSourceMapping_OnFailure struct {

	// Destination AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-eventsourcemapping-onfailure.html#cfn-lambda-eventsourcemapping-onfailure-destination
	Destination string `json:"Destination,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *EventSourceMapping_OnFailure) AWSCloudFormationType() string {
	return "AWS::Lambda::EventSourceMapping.OnFailure"
}
