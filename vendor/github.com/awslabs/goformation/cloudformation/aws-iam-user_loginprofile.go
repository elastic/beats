package cloudformation

// AWSIAMUser_LoginProfile AWS CloudFormation Resource (AWS::IAM::User.LoginProfile)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iam-user-loginprofile.html
type AWSIAMUser_LoginProfile struct {

	// Password AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iam-user-loginprofile.html#cfn-iam-user-loginprofile-password
	Password string `json:"Password,omitempty"`

	// PasswordResetRequired AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iam-user-loginprofile.html#cfn-iam-user-loginprofile-passwordresetrequired
	PasswordResetRequired bool `json:"PasswordResetRequired,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSIAMUser_LoginProfile) AWSCloudFormationType() string {
	return "AWS::IAM::User.LoginProfile"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSIAMUser_LoginProfile) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
