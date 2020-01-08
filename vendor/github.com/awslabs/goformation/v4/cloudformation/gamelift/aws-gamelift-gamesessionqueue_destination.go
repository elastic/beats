package gamelift

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// GameSessionQueue_Destination AWS CloudFormation Resource (AWS::GameLift::GameSessionQueue.Destination)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-gamelift-gamesessionqueue-destination.html
type GameSessionQueue_Destination struct {

	// DestinationArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-gamelift-gamesessionqueue-destination.html#cfn-gamelift-gamesessionqueue-destination-destinationarn
	DestinationArn string `json:"DestinationArn,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *GameSessionQueue_Destination) AWSCloudFormationType() string {
	return "AWS::GameLift::GameSessionQueue.Destination"
}
