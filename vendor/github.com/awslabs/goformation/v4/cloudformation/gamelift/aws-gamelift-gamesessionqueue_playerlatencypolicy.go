package gamelift

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// GameSessionQueue_PlayerLatencyPolicy AWS CloudFormation Resource (AWS::GameLift::GameSessionQueue.PlayerLatencyPolicy)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-gamelift-gamesessionqueue-playerlatencypolicy.html
type GameSessionQueue_PlayerLatencyPolicy struct {

	// MaximumIndividualPlayerLatencyMilliseconds AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-gamelift-gamesessionqueue-playerlatencypolicy.html#cfn-gamelift-gamesessionqueue-playerlatencypolicy-maximumindividualplayerlatencymilliseconds
	MaximumIndividualPlayerLatencyMilliseconds int `json:"MaximumIndividualPlayerLatencyMilliseconds,omitempty"`

	// PolicyDurationSeconds AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-gamelift-gamesessionqueue-playerlatencypolicy.html#cfn-gamelift-gamesessionqueue-playerlatencypolicy-policydurationseconds
	PolicyDurationSeconds int `json:"PolicyDurationSeconds,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *GameSessionQueue_PlayerLatencyPolicy) AWSCloudFormationType() string {
	return "AWS::GameLift::GameSessionQueue.PlayerLatencyPolicy"
}
