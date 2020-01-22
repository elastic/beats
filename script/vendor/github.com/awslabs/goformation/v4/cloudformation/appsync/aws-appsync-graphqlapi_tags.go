package appsync

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// GraphQLApi_Tags AWS CloudFormation Resource (AWS::AppSync::GraphQLApi.Tags)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-graphqlapi-tags.html
type GraphQLApi_Tags struct {

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *GraphQLApi_Tags) AWSCloudFormationType() string {
	return "AWS::AppSync::GraphQLApi.Tags"
}
