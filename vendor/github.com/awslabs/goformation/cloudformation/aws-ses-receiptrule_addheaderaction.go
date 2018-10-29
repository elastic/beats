package cloudformation

// AWSSESReceiptRule_AddHeaderAction AWS CloudFormation Resource (AWS::SES::ReceiptRule.AddHeaderAction)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptrule-addheaderaction.html
type AWSSESReceiptRule_AddHeaderAction struct {

	// HeaderName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptrule-addheaderaction.html#cfn-ses-receiptrule-addheaderaction-headername
	HeaderName string `json:"HeaderName,omitempty"`

	// HeaderValue AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-receiptrule-addheaderaction.html#cfn-ses-receiptrule-addheaderaction-headervalue
	HeaderValue string `json:"HeaderValue,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSESReceiptRule_AddHeaderAction) AWSCloudFormationType() string {
	return "AWS::SES::ReceiptRule.AddHeaderAction"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSESReceiptRule_AddHeaderAction) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
