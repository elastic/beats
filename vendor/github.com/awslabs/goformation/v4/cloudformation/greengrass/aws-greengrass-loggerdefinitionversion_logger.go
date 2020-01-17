package greengrass

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// LoggerDefinitionVersion_Logger AWS CloudFormation Resource (AWS::Greengrass::LoggerDefinitionVersion.Logger)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-loggerdefinitionversion-logger.html
type LoggerDefinitionVersion_Logger struct {

	// Component AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-loggerdefinitionversion-logger.html#cfn-greengrass-loggerdefinitionversion-logger-component
	Component string `json:"Component,omitempty"`

	// Id AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-loggerdefinitionversion-logger.html#cfn-greengrass-loggerdefinitionversion-logger-id
	Id string `json:"Id,omitempty"`

	// Level AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-loggerdefinitionversion-logger.html#cfn-greengrass-loggerdefinitionversion-logger-level
	Level string `json:"Level,omitempty"`

	// Space AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-loggerdefinitionversion-logger.html#cfn-greengrass-loggerdefinitionversion-logger-space
	Space int `json:"Space,omitempty"`

	// Type AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-loggerdefinitionversion-logger.html#cfn-greengrass-loggerdefinitionversion-logger-type
	Type string `json:"Type,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *LoggerDefinitionVersion_Logger) AWSCloudFormationType() string {
	return "AWS::Greengrass::LoggerDefinitionVersion.Logger"
}
