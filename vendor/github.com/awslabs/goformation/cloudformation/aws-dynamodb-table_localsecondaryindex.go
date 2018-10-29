package cloudformation

// AWSDynamoDBTable_LocalSecondaryIndex AWS CloudFormation Resource (AWS::DynamoDB::Table.LocalSecondaryIndex)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-lsi.html
type AWSDynamoDBTable_LocalSecondaryIndex struct {

	// IndexName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-lsi.html#cfn-dynamodb-lsi-indexname
	IndexName string `json:"IndexName,omitempty"`

	// KeySchema AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-lsi.html#cfn-dynamodb-lsi-keyschema
	KeySchema []AWSDynamoDBTable_KeySchema `json:"KeySchema,omitempty"`

	// Projection AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-lsi.html#cfn-dynamodb-lsi-projection
	Projection *AWSDynamoDBTable_Projection `json:"Projection,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSDynamoDBTable_LocalSecondaryIndex) AWSCloudFormationType() string {
	return "AWS::DynamoDB::Table.LocalSecondaryIndex"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSDynamoDBTable_LocalSecondaryIndex) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
