package appsync

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// DataSource_RelationalDatabaseConfig AWS CloudFormation Resource (AWS::AppSync::DataSource.RelationalDatabaseConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-datasource-relationaldatabaseconfig.html
type DataSource_RelationalDatabaseConfig struct {

	// RdsHttpEndpointConfig AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-datasource-relationaldatabaseconfig.html#cfn-appsync-datasource-relationaldatabaseconfig-rdshttpendpointconfig
	RdsHttpEndpointConfig *DataSource_RdsHttpEndpointConfig `json:"RdsHttpEndpointConfig,omitempty"`

	// RelationalDatabaseSourceType AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-datasource-relationaldatabaseconfig.html#cfn-appsync-datasource-relationaldatabaseconfig-relationaldatabasesourcetype
	RelationalDatabaseSourceType string `json:"RelationalDatabaseSourceType,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *DataSource_RelationalDatabaseConfig) AWSCloudFormationType() string {
	return "AWS::AppSync::DataSource.RelationalDatabaseConfig"
}
