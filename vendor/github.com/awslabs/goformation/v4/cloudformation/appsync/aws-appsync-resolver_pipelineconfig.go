package appsync

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Resolver_PipelineConfig AWS CloudFormation Resource (AWS::AppSync::Resolver.PipelineConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-resolver-pipelineconfig.html
type Resolver_PipelineConfig struct {

	// Functions AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-resolver-pipelineconfig.html#cfn-appsync-resolver-pipelineconfig-functions
	Functions []string `json:"Functions,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Resolver_PipelineConfig) AWSCloudFormationType() string {
	return "AWS::AppSync::Resolver.PipelineConfig"
}
