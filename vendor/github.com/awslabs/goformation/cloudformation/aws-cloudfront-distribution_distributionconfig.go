package cloudformation

// AWSCloudFrontDistribution_DistributionConfig AWS CloudFormation Resource (AWS::CloudFront::Distribution.DistributionConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-distributionconfig.html
type AWSCloudFrontDistribution_DistributionConfig struct {

	// Aliases AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-distributionconfig.html#cfn-cloudfront-distribution-distributionconfig-aliases
	Aliases []string `json:"Aliases,omitempty"`

	// CacheBehaviors AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-distributionconfig.html#cfn-cloudfront-distribution-distributionconfig-cachebehaviors
	CacheBehaviors []AWSCloudFrontDistribution_CacheBehavior `json:"CacheBehaviors,omitempty"`

	// Comment AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-distributionconfig.html#cfn-cloudfront-distribution-distributionconfig-comment
	Comment string `json:"Comment,omitempty"`

	// CustomErrorResponses AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-distributionconfig.html#cfn-cloudfront-distribution-distributionconfig-customerrorresponses
	CustomErrorResponses []AWSCloudFrontDistribution_CustomErrorResponse `json:"CustomErrorResponses,omitempty"`

	// DefaultCacheBehavior AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-distributionconfig.html#cfn-cloudfront-distribution-distributionconfig-defaultcachebehavior
	DefaultCacheBehavior *AWSCloudFrontDistribution_DefaultCacheBehavior `json:"DefaultCacheBehavior,omitempty"`

	// DefaultRootObject AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-distributionconfig.html#cfn-cloudfront-distribution-distributionconfig-defaultrootobject
	DefaultRootObject string `json:"DefaultRootObject,omitempty"`

	// Enabled AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-distributionconfig.html#cfn-cloudfront-distribution-distributionconfig-enabled
	Enabled bool `json:"Enabled,omitempty"`

	// HttpVersion AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-distributionconfig.html#cfn-cloudfront-distribution-distributionconfig-httpversion
	HttpVersion string `json:"HttpVersion,omitempty"`

	// IPV6Enabled AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-distributionconfig.html#cfn-cloudfront-distribution-distributionconfig-ipv6enabled
	IPV6Enabled bool `json:"IPV6Enabled,omitempty"`

	// Logging AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-distributionconfig.html#cfn-cloudfront-distribution-distributionconfig-logging
	Logging *AWSCloudFrontDistribution_Logging `json:"Logging,omitempty"`

	// Origins AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-distributionconfig.html#cfn-cloudfront-distribution-distributionconfig-origins
	Origins []AWSCloudFrontDistribution_Origin `json:"Origins,omitempty"`

	// PriceClass AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-distributionconfig.html#cfn-cloudfront-distribution-distributionconfig-priceclass
	PriceClass string `json:"PriceClass,omitempty"`

	// Restrictions AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-distributionconfig.html#cfn-cloudfront-distribution-distributionconfig-restrictions
	Restrictions *AWSCloudFrontDistribution_Restrictions `json:"Restrictions,omitempty"`

	// ViewerCertificate AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-distributionconfig.html#cfn-cloudfront-distribution-distributionconfig-viewercertificate
	ViewerCertificate *AWSCloudFrontDistribution_ViewerCertificate `json:"ViewerCertificate,omitempty"`

	// WebACLId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-distributionconfig.html#cfn-cloudfront-distribution-distributionconfig-webaclid
	WebACLId string `json:"WebACLId,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCloudFrontDistribution_DistributionConfig) AWSCloudFormationType() string {
	return "AWS::CloudFront::Distribution.DistributionConfig"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCloudFrontDistribution_DistributionConfig) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
