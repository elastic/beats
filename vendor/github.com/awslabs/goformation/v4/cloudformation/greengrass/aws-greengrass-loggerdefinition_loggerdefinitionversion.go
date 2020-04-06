package greengrass

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// LoggerDefinition_LoggerDefinitionVersion AWS CloudFormation Resource (AWS::Greengrass::LoggerDefinition.LoggerDefinitionVersion)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-loggerdefinition-loggerdefinitionversion.html
type LoggerDefinition_LoggerDefinitionVersion struct {

	// Loggers AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-loggerdefinition-loggerdefinitionversion.html#cfn-greengrass-loggerdefinition-loggerdefinitionversion-loggers
	Loggers []LoggerDefinition_Logger `json:"Loggers,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *LoggerDefinition_LoggerDefinitionVersion) AWSCloudFormationType() string {
	return "AWS::Greengrass::LoggerDefinition.LoggerDefinitionVersion"
}
