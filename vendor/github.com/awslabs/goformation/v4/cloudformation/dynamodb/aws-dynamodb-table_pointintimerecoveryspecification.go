package dynamodb

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Table_PointInTimeRecoverySpecification AWS CloudFormation Resource (AWS::DynamoDB::Table.PointInTimeRecoverySpecification)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-table-pointintimerecoveryspecification.html
type Table_PointInTimeRecoverySpecification struct {

	// PointInTimeRecoveryEnabled AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-table-pointintimerecoveryspecification.html#cfn-dynamodb-table-pointintimerecoveryspecification-pointintimerecoveryenabled
	PointInTimeRecoveryEnabled bool `json:"PointInTimeRecoveryEnabled,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Table_PointInTimeRecoverySpecification) AWSCloudFormationType() string {
	return "AWS::DynamoDB::Table.PointInTimeRecoverySpecification"
}
