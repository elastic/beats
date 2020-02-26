package gamelift

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Fleet_ServerProcess AWS CloudFormation Resource (AWS::GameLift::Fleet.ServerProcess)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-gamelift-fleet-serverprocess.html
type Fleet_ServerProcess struct {

	// ConcurrentExecutions AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-gamelift-fleet-serverprocess.html#cfn-gamelift-fleet-serverprocess-concurrentexecutions
	ConcurrentExecutions int `json:"ConcurrentExecutions"`

	// LaunchPath AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-gamelift-fleet-serverprocess.html#cfn-gamelift-fleet-serverprocess-launchpath
	LaunchPath string `json:"LaunchPath,omitempty"`

	// Parameters AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-gamelift-fleet-serverprocess.html#cfn-gamelift-fleet-serverprocess-parameters
	Parameters string `json:"Parameters,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Fleet_ServerProcess) AWSCloudFormationType() string {
	return "AWS::GameLift::Fleet.ServerProcess"
}
