package appsync

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// GraphQLApi_AdditionalAuthenticationProvider AWS CloudFormation Resource (AWS::AppSync::GraphQLApi.AdditionalAuthenticationProvider)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-graphqlapi-additionalauthenticationprovider.html
type GraphQLApi_AdditionalAuthenticationProvider struct {

	// AuthenticationType AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-graphqlapi-additionalauthenticationprovider.html#cfn-appsync-graphqlapi-additionalauthenticationprovider-authenticationtype
	AuthenticationType string `json:"AuthenticationType,omitempty"`

	// OpenIDConnectConfig AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-graphqlapi-additionalauthenticationprovider.html#cfn-appsync-graphqlapi-additionalauthenticationprovider-openidconnectconfig
	OpenIDConnectConfig *GraphQLApi_OpenIDConnectConfig `json:"OpenIDConnectConfig,omitempty"`

	// UserPoolConfig AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-graphqlapi-additionalauthenticationprovider.html#cfn-appsync-graphqlapi-additionalauthenticationprovider-userpoolconfig
	UserPoolConfig *GraphQLApi_CognitoUserPoolConfig `json:"UserPoolConfig,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *GraphQLApi_AdditionalAuthenticationProvider) AWSCloudFormationType() string {
	return "AWS::AppSync::GraphQLApi.AdditionalAuthenticationProvider"
}
