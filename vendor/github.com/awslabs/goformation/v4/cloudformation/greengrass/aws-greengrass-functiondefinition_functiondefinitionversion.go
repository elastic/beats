package greengrass

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// FunctionDefinition_FunctionDefinitionVersion AWS CloudFormation Resource (AWS::Greengrass::FunctionDefinition.FunctionDefinitionVersion)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-functiondefinition-functiondefinitionversion.html
type FunctionDefinition_FunctionDefinitionVersion struct {

	// DefaultConfig AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-functiondefinition-functiondefinitionversion.html#cfn-greengrass-functiondefinition-functiondefinitionversion-defaultconfig
	DefaultConfig *FunctionDefinition_DefaultConfig `json:"DefaultConfig,omitempty"`

	// Functions AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-functiondefinition-functiondefinitionversion.html#cfn-greengrass-functiondefinition-functiondefinitionversion-functions
	Functions []FunctionDefinition_Function `json:"Functions,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *FunctionDefinition_FunctionDefinitionVersion) AWSCloudFormationType() string {
	return "AWS::Greengrass::FunctionDefinition.FunctionDefinitionVersion"
}
