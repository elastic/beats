package cloudformation

// AWSDynamoDBTable_GlobalSecondaryIndex AWS CloudFormation Resource (AWS::DynamoDB::Table.GlobalSecondaryIndex)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-gsi.html
type AWSDynamoDBTable_GlobalSecondaryIndex struct {

	// IndexName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-gsi.html#cfn-dynamodb-gsi-indexname
	IndexName string `json:"IndexName,omitempty"`

	// KeySchema AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-gsi.html#cfn-dynamodb-gsi-keyschema
	KeySchema []AWSDynamoDBTable_KeySchema `json:"KeySchema,omitempty"`

	// Projection AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-gsi.html#cfn-dynamodb-gsi-projection
	Projection *AWSDynamoDBTable_Projection `json:"Projection,omitempty"`

	// ProvisionedThroughput AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-gsi.html#cfn-dynamodb-gsi-provisionedthroughput
	ProvisionedThroughput *AWSDynamoDBTable_ProvisionedThroughput `json:"ProvisionedThroughput,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSDynamoDBTable_GlobalSecondaryIndex) AWSCloudFormationType() string {
	return "AWS::DynamoDB::Table.GlobalSecondaryIndex"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSDynamoDBTable_GlobalSecondaryIndex) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
