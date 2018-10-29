package cloudformation

// AWSCloudFrontCloudFrontOriginAccessIdentity_CloudFrontOriginAccessIdentityConfig AWS CloudFormation Resource (AWS::CloudFront::CloudFrontOriginAccessIdentity.CloudFrontOriginAccessIdentityConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-cloudfrontoriginaccessidentity-cloudfrontoriginaccessidentityconfig.html
type AWSCloudFrontCloudFrontOriginAccessIdentity_CloudFrontOriginAccessIdentityConfig struct {

	// Comment AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-cloudfrontoriginaccessidentity-cloudfrontoriginaccessidentityconfig.html#cfn-cloudfront-cloudfrontoriginaccessidentity-cloudfrontoriginaccessidentityconfig-comment
	Comment string `json:"Comment,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCloudFrontCloudFrontOriginAccessIdentity_CloudFrontOriginAccessIdentityConfig) AWSCloudFormationType() string {
	return "AWS::CloudFront::CloudFrontOriginAccessIdentity.CloudFrontOriginAccessIdentityConfig"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCloudFrontCloudFrontOriginAccessIdentity_CloudFrontOriginAccessIdentityConfig) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
