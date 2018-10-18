package cloudformation

// AWSDynamoDBTable_SSESpecification AWS CloudFormation Resource (AWS::DynamoDB::Table.SSESpecification)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-table-ssespecification.html
type AWSDynamoDBTable_SSESpecification struct {

	// SSEEnabled AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-table-ssespecification.html#cfn-dynamodb-table-ssespecification-sseenabled
	SSEEnabled bool `json:"SSEEnabled,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSDynamoDBTable_SSESpecification) AWSCloudFormationType() string {
	return "AWS::DynamoDB::Table.SSESpecification"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSDynamoDBTable_SSESpecification) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
