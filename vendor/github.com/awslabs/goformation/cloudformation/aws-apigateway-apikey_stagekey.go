package cloudformation

// AWSApiGatewayApiKey_StageKey AWS CloudFormation Resource (AWS::ApiGateway::ApiKey.StageKey)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-apigateway-apikey-stagekey.html
type AWSApiGatewayApiKey_StageKey struct {

	// RestApiId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-apigateway-apikey-stagekey.html#cfn-apigateway-apikey-stagekey-restapiid
	RestApiId string `json:"RestApiId,omitempty"`

	// StageName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-apigateway-apikey-stagekey.html#cfn-apigateway-apikey-stagekey-stagename
	StageName string `json:"StageName,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSApiGatewayApiKey_StageKey) AWSCloudFormationType() string {
	return "AWS::ApiGateway::ApiKey.StageKey"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSApiGatewayApiKey_StageKey) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
