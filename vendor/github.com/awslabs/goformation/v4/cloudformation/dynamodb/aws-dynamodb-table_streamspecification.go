package dynamodb

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Table_StreamSpecification AWS CloudFormation Resource (AWS::DynamoDB::Table.StreamSpecification)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-streamspecification.html
type Table_StreamSpecification struct {

	// StreamViewType AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-streamspecification.html#cfn-dynamodb-streamspecification-streamviewtype
	StreamViewType string `json:"StreamViewType,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Table_StreamSpecification) AWSCloudFormationType() string {
	return "AWS::DynamoDB::Table.StreamSpecification"
}
