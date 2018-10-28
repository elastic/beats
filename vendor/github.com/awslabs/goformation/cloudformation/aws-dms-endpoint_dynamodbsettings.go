package cloudformation

// AWSDMSEndpoint_DynamoDbSettings AWS CloudFormation Resource (AWS::DMS::Endpoint.DynamoDbSettings)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dms-endpoint-dynamodbsettings.html
type AWSDMSEndpoint_DynamoDbSettings struct {

	// ServiceAccessRoleArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dms-endpoint-dynamodbsettings.html#cfn-dms-endpoint-dynamodbsettings-serviceaccessrolearn
	ServiceAccessRoleArn string `json:"ServiceAccessRoleArn,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSDMSEndpoint_DynamoDbSettings) AWSCloudFormationType() string {
	return "AWS::DMS::Endpoint.DynamoDbSettings"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSDMSEndpoint_DynamoDbSettings) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
