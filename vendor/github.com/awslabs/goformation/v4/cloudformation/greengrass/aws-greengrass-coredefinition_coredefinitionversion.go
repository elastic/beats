package greengrass

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// CoreDefinition_CoreDefinitionVersion AWS CloudFormation Resource (AWS::Greengrass::CoreDefinition.CoreDefinitionVersion)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-coredefinition-coredefinitionversion.html
type CoreDefinition_CoreDefinitionVersion struct {

	// Cores AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-coredefinition-coredefinitionversion.html#cfn-greengrass-coredefinition-coredefinitionversion-cores
	Cores []CoreDefinition_Core `json:"Cores,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *CoreDefinition_CoreDefinitionVersion) AWSCloudFormationType() string {
	return "AWS::Greengrass::CoreDefinition.CoreDefinitionVersion"
}
