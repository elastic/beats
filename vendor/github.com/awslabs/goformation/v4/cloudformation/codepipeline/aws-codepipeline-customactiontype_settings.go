package codepipeline

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// CustomActionType_Settings AWS CloudFormation Resource (AWS::CodePipeline::CustomActionType.Settings)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-customactiontype-settings.html
type CustomActionType_Settings struct {

	// EntityUrlTemplate AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-customactiontype-settings.html#cfn-codepipeline-customactiontype-settings-entityurltemplate
	EntityUrlTemplate string `json:"EntityUrlTemplate,omitempty"`

	// ExecutionUrlTemplate AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-customactiontype-settings.html#cfn-codepipeline-customactiontype-settings-executionurltemplate
	ExecutionUrlTemplate string `json:"ExecutionUrlTemplate,omitempty"`

	// RevisionUrlTemplate AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-customactiontype-settings.html#cfn-codepipeline-customactiontype-settings-revisionurltemplate
	RevisionUrlTemplate string `json:"RevisionUrlTemplate,omitempty"`

	// ThirdPartyConfigurationUrl AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-customactiontype-settings.html#cfn-codepipeline-customactiontype-settings-thirdpartyconfigurationurl
	ThirdPartyConfigurationUrl string `json:"ThirdPartyConfigurationUrl,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *CustomActionType_Settings) AWSCloudFormationType() string {
	return "AWS::CodePipeline::CustomActionType.Settings"
}
