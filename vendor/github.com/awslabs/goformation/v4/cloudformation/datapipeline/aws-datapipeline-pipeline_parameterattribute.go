package datapipeline

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Pipeline_ParameterAttribute AWS CloudFormation Resource (AWS::DataPipeline::Pipeline.ParameterAttribute)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-datapipeline-pipeline-parameterobjects-attributes.html
type Pipeline_ParameterAttribute struct {

	// Key AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-datapipeline-pipeline-parameterobjects-attributes.html#cfn-datapipeline-pipeline-parameterobjects-attribtues-key
	Key string `json:"Key,omitempty"`

	// StringValue AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-datapipeline-pipeline-parameterobjects-attributes.html#cfn-datapipeline-pipeline-parameterobjects-attribtues-stringvalue
	StringValue string `json:"StringValue,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Pipeline_ParameterAttribute) AWSCloudFormationType() string {
	return "AWS::DataPipeline::Pipeline.ParameterAttribute"
}
