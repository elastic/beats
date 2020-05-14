package sagemaker

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Endpoint_VariantProperty AWS CloudFormation Resource (AWS::SageMaker::Endpoint.VariantProperty)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-sagemaker-endpoint-variantproperty.html
type Endpoint_VariantProperty struct {

	// VariantPropertyType AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-sagemaker-endpoint-variantproperty.html#cfn-sagemaker-endpoint-variantproperty-variantpropertytype
	VariantPropertyType string `json:"VariantPropertyType,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Endpoint_VariantProperty) AWSCloudFormationType() string {
	return "AWS::SageMaker::Endpoint.VariantProperty"
}
