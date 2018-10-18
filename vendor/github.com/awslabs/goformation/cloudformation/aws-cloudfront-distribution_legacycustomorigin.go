package cloudformation

// AWSCloudFrontDistribution_LegacyCustomOrigin AWS CloudFormation Resource (AWS::CloudFront::Distribution.LegacyCustomOrigin)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-legacycustomorigin.html
type AWSCloudFrontDistribution_LegacyCustomOrigin struct {

	// DNSName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-legacycustomorigin.html#cfn-cloudfront-distribution-legacycustomorigin-dnsname
	DNSName string `json:"DNSName,omitempty"`

	// HTTPPort AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-legacycustomorigin.html#cfn-cloudfront-distribution-legacycustomorigin-httpport
	HTTPPort int `json:"HTTPPort,omitempty"`

	// HTTPSPort AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-legacycustomorigin.html#cfn-cloudfront-distribution-legacycustomorigin-httpsport
	HTTPSPort int `json:"HTTPSPort,omitempty"`

	// OriginProtocolPolicy AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-legacycustomorigin.html#cfn-cloudfront-distribution-legacycustomorigin-originprotocolpolicy
	OriginProtocolPolicy string `json:"OriginProtocolPolicy,omitempty"`

	// OriginSSLProtocols AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-legacycustomorigin.html#cfn-cloudfront-distribution-legacycustomorigin-originsslprotocols
	OriginSSLProtocols []string `json:"OriginSSLProtocols,omitempty"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCloudFrontDistribution_LegacyCustomOrigin) AWSCloudFormationType() string {
	return "AWS::CloudFront::Distribution.LegacyCustomOrigin"
}
