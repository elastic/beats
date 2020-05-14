package appsync

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Resolver_LambdaConflictHandlerConfig AWS CloudFormation Resource (AWS::AppSync::Resolver.LambdaConflictHandlerConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-resolver-lambdaconflicthandlerconfig.html
type Resolver_LambdaConflictHandlerConfig struct {

	// LambdaConflictHandlerArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-resolver-lambdaconflicthandlerconfig.html#cfn-appsync-resolver-lambdaconflicthandlerconfig-lambdaconflicthandlerarn
	LambdaConflictHandlerArn string `json:"LambdaConflictHandlerArn,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Resolver_LambdaConflictHandlerConfig) AWSCloudFormationType() string {
	return "AWS::AppSync::Resolver.LambdaConflictHandlerConfig"
}
