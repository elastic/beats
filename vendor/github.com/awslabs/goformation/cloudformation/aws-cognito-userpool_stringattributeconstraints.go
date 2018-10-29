package cloudformation

// AWSCognitoUserPool_StringAttributeConstraints AWS CloudFormation Resource (AWS::Cognito::UserPool.StringAttributeConstraints)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-stringattributeconstraints.html
type AWSCognitoUserPool_StringAttributeConstraints struct {

	// MaxLength AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-stringattributeconstraints.html#cfn-cognito-userpool-stringattributeconstraints-maxlength
	MaxLength string `json:"MaxLength,omitempty"`

	// MinLength AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-stringattributeconstraints.html#cfn-cognito-userpool-stringattributeconstraints-minlength
	MinLength string `json:"MinLength,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCognitoUserPool_StringAttributeConstraints) AWSCloudFormationType() string {
	return "AWS::Cognito::UserPool.StringAttributeConstraints"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCognitoUserPool_StringAttributeConstraints) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
