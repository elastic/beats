package cloudformation

// AWSCloudFrontDistribution_CustomOriginConfig AWS CloudFormation Resource (AWS::CloudFront::Distribution.CustomOriginConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-customoriginconfig.html
type AWSCloudFrontDistribution_CustomOriginConfig struct {

	// HTTPPort AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-customoriginconfig.html#cfn-cloudfront-distribution-customoriginconfig-httpport
	HTTPPort int `json:"HTTPPort,omitempty"`

	// HTTPSPort AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-customoriginconfig.html#cfn-cloudfront-distribution-customoriginconfig-httpsport
	HTTPSPort int `json:"HTTPSPort,omitempty"`

	// OriginKeepaliveTimeout AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-customoriginconfig.html#cfn-cloudfront-distribution-customoriginconfig-originkeepalivetimeout
	OriginKeepaliveTimeout int `json:"OriginKeepaliveTimeout,omitempty"`

	// OriginProtocolPolicy AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-customoriginconfig.html#cfn-cloudfront-distribution-customoriginconfig-originprotocolpolicy
	OriginProtocolPolicy string `json:"OriginProtocolPolicy,omitempty"`

	// OriginReadTimeout AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-customoriginconfig.html#cfn-cloudfront-distribution-customoriginconfig-originreadtimeout
	OriginReadTimeout int `json:"OriginReadTimeout,omitempty"`

	// OriginSSLProtocols AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-customoriginconfig.html#cfn-cloudfront-distribution-customoriginconfig-originsslprotocols
	OriginSSLProtocols []string `json:"OriginSSLProtocols,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCloudFrontDistribution_CustomOriginConfig) AWSCloudFormationType() string {
	return "AWS::CloudFront::Distribution.CustomOriginConfig"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCloudFrontDistribution_CustomOriginConfig) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
