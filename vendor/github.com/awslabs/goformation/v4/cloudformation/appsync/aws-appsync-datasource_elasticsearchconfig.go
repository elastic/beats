package appsync

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// DataSource_ElasticsearchConfig AWS CloudFormation Resource (AWS::AppSync::DataSource.ElasticsearchConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-datasource-elasticsearchconfig.html
type DataSource_ElasticsearchConfig struct {

	// AwsRegion AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-datasource-elasticsearchconfig.html#cfn-appsync-datasource-elasticsearchconfig-awsregion
	AwsRegion string `json:"AwsRegion,omitempty"`

	// Endpoint AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-datasource-elasticsearchconfig.html#cfn-appsync-datasource-elasticsearchconfig-endpoint
	Endpoint string `json:"Endpoint,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *DataSource_ElasticsearchConfig) AWSCloudFormationType() string {
	return "AWS::AppSync::DataSource.ElasticsearchConfig"
}
