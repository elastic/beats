package cloudformation

// AWSDynamoDBTable_Projection AWS CloudFormation Resource (AWS::DynamoDB::Table.Projection)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-projectionobject.html
type AWSDynamoDBTable_Projection struct {

	// NonKeyAttributes AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-projectionobject.html#cfn-dynamodb-projectionobj-nonkeyatt
	NonKeyAttributes []string `json:"NonKeyAttributes,omitempty"`

	// ProjectionType AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dynamodb-projectionobject.html#cfn-dynamodb-projectionobj-projtype
	ProjectionType string `json:"ProjectionType,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSDynamoDBTable_Projection) AWSCloudFormationType() string {
	return "AWS::DynamoDB::Table.Projection"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSDynamoDBTable_Projection) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
