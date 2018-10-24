package cloudformation

// AWSCloudFrontDistribution_LegacyS3Origin AWS CloudFormation Resource (AWS::CloudFront::Distribution.LegacyS3Origin)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-legacys3origin.html
type AWSCloudFrontDistribution_LegacyS3Origin struct {

	// DNSName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-legacys3origin.html#cfn-cloudfront-distribution-legacys3origin-dnsname
	DNSName string `json:"DNSName,omitempty"`

	// OriginAccessIdentity AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-legacys3origin.html#cfn-cloudfront-distribution-legacys3origin-originaccessidentity
	OriginAccessIdentity string `json:"OriginAccessIdentity,omitempty"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCloudFrontDistribution_LegacyS3Origin) AWSCloudFormationType() string {
	return "AWS::CloudFront::Distribution.LegacyS3Origin"
}
