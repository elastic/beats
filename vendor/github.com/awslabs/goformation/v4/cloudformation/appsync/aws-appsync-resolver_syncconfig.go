package appsync

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Resolver_SyncConfig AWS CloudFormation Resource (AWS::AppSync::Resolver.SyncConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-resolver-syncconfig.html
type Resolver_SyncConfig struct {

	// ConflictDetection AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-resolver-syncconfig.html#cfn-appsync-resolver-syncconfig-conflictdetection
	ConflictDetection string `json:"ConflictDetection,omitempty"`

	// ConflictHandler AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-resolver-syncconfig.html#cfn-appsync-resolver-syncconfig-conflicthandler
	ConflictHandler string `json:"ConflictHandler,omitempty"`

	// LambdaConflictHandlerConfig AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-resolver-syncconfig.html#cfn-appsync-resolver-syncconfig-lambdaconflicthandlerconfig
	LambdaConflictHandlerConfig *Resolver_LambdaConflictHandlerConfig `json:"LambdaConflictHandlerConfig,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Resolver_SyncConfig) AWSCloudFormationType() string {
	return "AWS::AppSync::Resolver.SyncConfig"
}
