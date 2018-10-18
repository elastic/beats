package cloudformation

// AWSCognitoUserPool_DeviceConfiguration AWS CloudFormation Resource (AWS::Cognito::UserPool.DeviceConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-deviceconfiguration.html
type AWSCognitoUserPool_DeviceConfiguration struct {

	// ChallengeRequiredOnNewDevice AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-deviceconfiguration.html#cfn-cognito-userpool-deviceconfiguration-challengerequiredonnewdevice
	ChallengeRequiredOnNewDevice bool `json:"ChallengeRequiredOnNewDevice,omitempty"`

	// DeviceOnlyRememberedOnUserPrompt AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-deviceconfiguration.html#cfn-cognito-userpool-deviceconfiguration-deviceonlyrememberedonuserprompt
	DeviceOnlyRememberedOnUserPrompt bool `json:"DeviceOnlyRememberedOnUserPrompt,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCognitoUserPool_DeviceConfiguration) AWSCloudFormationType() string {
	return "AWS::Cognito::UserPool.DeviceConfiguration"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCognitoUserPool_DeviceConfiguration) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
