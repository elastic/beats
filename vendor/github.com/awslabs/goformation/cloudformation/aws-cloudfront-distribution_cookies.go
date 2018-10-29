package cloudformation

// AWSCloudFrontDistribution_Cookies AWS CloudFormation Resource (AWS::CloudFront::Distribution.Cookies)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-cookies.html
type AWSCloudFrontDistribution_Cookies struct {

	// Forward AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-cookies.html#cfn-cloudfront-distribution-cookies-forward
	Forward string `json:"Forward,omitempty"`

	// WhitelistedNames AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-cookies.html#cfn-cloudfront-distribution-cookies-whitelistednames
	WhitelistedNames []string `json:"WhitelistedNames,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCloudFrontDistribution_Cookies) AWSCloudFormationType() string {
	return "AWS::CloudFront::Distribution.Cookies"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCloudFrontDistribution_Cookies) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
