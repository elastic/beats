package serverless

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// SimpleTable_SSESpecification AWS CloudFormation Resource (AWS::Serverless::SimpleTable.SSESpecification)
// See: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-table-ssespecification.html
type SimpleTable_SSESpecification struct {

	// SSEEnabled AWS CloudFormation Property
	// Required: false
	// See: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-table-ssespecification.html
	SSEEnabled bool `json:"SSEEnabled,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *SimpleTable_SSESpecification) AWSCloudFormationType() string {
	return "AWS::Serverless::SimpleTable.SSESpecification"
}
