package cloudformation

// AWSCloudFrontStreamingDistribution_S3Origin AWS CloudFormation Resource (AWS::CloudFront::StreamingDistribution.S3Origin)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-streamingdistribution-s3origin.html
type AWSCloudFrontStreamingDistribution_S3Origin struct {

	// DomainName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-streamingdistribution-s3origin.html#cfn-cloudfront-streamingdistribution-s3origin-domainname
	DomainName string `json:"DomainName,omitempty"`

	// OriginAccessIdentity AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-streamingdistribution-s3origin.html#cfn-cloudfront-streamingdistribution-s3origin-originaccessidentity
	OriginAccessIdentity string `json:"OriginAccessIdentity,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCloudFrontStreamingDistribution_S3Origin) AWSCloudFormationType() string {
	return "AWS::CloudFront::StreamingDistribution.S3Origin"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCloudFrontStreamingDistribution_S3Origin) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
