package cloudformation

// AWSCognitoUserPool_InviteMessageTemplate AWS CloudFormation Resource (AWS::Cognito::UserPool.InviteMessageTemplate)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-invitemessagetemplate.html
type AWSCognitoUserPool_InviteMessageTemplate struct {

	// EmailMessage AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-invitemessagetemplate.html#cfn-cognito-userpool-invitemessagetemplate-emailmessage
	EmailMessage string `json:"EmailMessage,omitempty"`

	// EmailSubject AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-invitemessagetemplate.html#cfn-cognito-userpool-invitemessagetemplate-emailsubject
	EmailSubject string `json:"EmailSubject,omitempty"`

	// SMSMessage AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-invitemessagetemplate.html#cfn-cognito-userpool-invitemessagetemplate-smsmessage
	SMSMessage string `json:"SMSMessage,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCognitoUserPool_InviteMessageTemplate) AWSCloudFormationType() string {
	return "AWS::Cognito::UserPool.InviteMessageTemplate"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCognitoUserPool_InviteMessageTemplate) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
