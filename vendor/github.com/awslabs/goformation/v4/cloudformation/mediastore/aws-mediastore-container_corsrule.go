package mediastore

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Container_CorsRule AWS CloudFormation Resource (AWS::MediaStore::Container.CorsRule)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-mediastore-container-corsrule.html
type Container_CorsRule struct {

	// AllowedHeaders AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-mediastore-container-corsrule.html#cfn-mediastore-container-corsrule-allowedheaders
	AllowedHeaders []string `json:"AllowedHeaders,omitempty"`

	// AllowedMethods AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-mediastore-container-corsrule.html#cfn-mediastore-container-corsrule-allowedmethods
	AllowedMethods []string `json:"AllowedMethods,omitempty"`

	// AllowedOrigins AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-mediastore-container-corsrule.html#cfn-mediastore-container-corsrule-allowedorigins
	AllowedOrigins []string `json:"AllowedOrigins,omitempty"`

	// ExposeHeaders AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-mediastore-container-corsrule.html#cfn-mediastore-container-corsrule-exposeheaders
	ExposeHeaders []string `json:"ExposeHeaders,omitempty"`

	// MaxAgeSeconds AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-mediastore-container-corsrule.html#cfn-mediastore-container-corsrule-maxageseconds
	MaxAgeSeconds int `json:"MaxAgeSeconds,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Container_CorsRule) AWSCloudFormationType() string {
	return "AWS::MediaStore::Container.CorsRule"
}
