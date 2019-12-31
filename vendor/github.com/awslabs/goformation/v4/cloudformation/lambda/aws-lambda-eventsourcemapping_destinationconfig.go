package lambda

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// EventSourceMapping_DestinationConfig AWS CloudFormation Resource (AWS::Lambda::EventSourceMapping.DestinationConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-eventsourcemapping-destinationconfig.html
type EventSourceMapping_DestinationConfig struct {

	// OnFailure AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-eventsourcemapping-destinationconfig.html#cfn-lambda-eventsourcemapping-destinationconfig-onfailure
	OnFailure *EventSourceMapping_OnFailure `json:"OnFailure,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *EventSourceMapping_DestinationConfig) AWSCloudFormationType() string {
	return "AWS::Lambda::EventSourceMapping.DestinationConfig"
}
