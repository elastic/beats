package appsync

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// GraphQLApi_LogConfig AWS CloudFormation Resource (AWS::AppSync::GraphQLApi.LogConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-graphqlapi-logconfig.html
type GraphQLApi_LogConfig struct {

	// CloudWatchLogsRoleArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-graphqlapi-logconfig.html#cfn-appsync-graphqlapi-logconfig-cloudwatchlogsrolearn
	CloudWatchLogsRoleArn string `json:"CloudWatchLogsRoleArn,omitempty"`

	// ExcludeVerboseContent AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-graphqlapi-logconfig.html#cfn-appsync-graphqlapi-logconfig-excludeverbosecontent
	ExcludeVerboseContent bool `json:"ExcludeVerboseContent,omitempty"`

	// FieldLogLevel AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-graphqlapi-logconfig.html#cfn-appsync-graphqlapi-logconfig-fieldloglevel
	FieldLogLevel string `json:"FieldLogLevel,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *GraphQLApi_LogConfig) AWSCloudFormationType() string {
	return "AWS::AppSync::GraphQLApi.LogConfig"
}
