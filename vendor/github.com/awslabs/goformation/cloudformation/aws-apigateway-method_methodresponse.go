package cloudformation

// AWSApiGatewayMethod_MethodResponse AWS CloudFormation Resource (AWS::ApiGateway::Method.MethodResponse)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-apitgateway-method-methodresponse.html
type AWSApiGatewayMethod_MethodResponse struct {

	// ResponseModels AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-apitgateway-method-methodresponse.html#cfn-apigateway-method-methodresponse-responsemodels
	ResponseModels map[string]string `json:"ResponseModels,omitempty"`

	// ResponseParameters AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-apitgateway-method-methodresponse.html#cfn-apigateway-method-methodresponse-responseparameters
	ResponseParameters map[string]bool `json:"ResponseParameters,omitempty"`

	// StatusCode AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-apitgateway-method-methodresponse.html#cfn-apigateway-method-methodresponse-statuscode
	StatusCode string `json:"StatusCode,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSApiGatewayMethod_MethodResponse) AWSCloudFormationType() string {
	return "AWS::ApiGateway::Method.MethodResponse"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSApiGatewayMethod_MethodResponse) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
