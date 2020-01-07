package serverless

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Api_CorsConfiguration AWS CloudFormation Resource (AWS::Serverless::Api.CorsConfiguration)
// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#cors-configuration
type Api_CorsConfiguration struct {

	// AllowCredentials AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#cors-configuration
	AllowCredentials bool `json:"AllowCredentials,omitempty"`

	// AllowHeaders AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#cors-configuration
	AllowHeaders string `json:"AllowHeaders,omitempty"`

	// AllowMethods AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#cors-configuration
	AllowMethods string `json:"AllowMethods,omitempty"`

	// AllowOrigin AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#cors-configuration
	AllowOrigin string `json:"AllowOrigin,omitempty"`

	// MaxAge AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#cors-configuration
	MaxAge string `json:"MaxAge,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Api_CorsConfiguration) AWSCloudFormationType() string {
	return "AWS::Serverless::Api.CorsConfiguration"
}
