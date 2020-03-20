package datapipeline

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Pipeline_ParameterValue AWS CloudFormation Resource (AWS::DataPipeline::Pipeline.ParameterValue)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-datapipeline-pipeline-parametervalues.html
type Pipeline_ParameterValue struct {

	// Id AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-datapipeline-pipeline-parametervalues.html#cfn-datapipeline-pipeline-parametervalues-id
	Id string `json:"Id,omitempty"`

	// StringValue AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-datapipeline-pipeline-parametervalues.html#cfn-datapipeline-pipeline-parametervalues-stringvalue
	StringValue string `json:"StringValue,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Pipeline_ParameterValue) AWSCloudFormationType() string {
	return "AWS::DataPipeline::Pipeline.ParameterValue"
}
