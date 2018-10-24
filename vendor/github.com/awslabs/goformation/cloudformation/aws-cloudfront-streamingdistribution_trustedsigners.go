package cloudformation

// AWSCloudFrontStreamingDistribution_TrustedSigners AWS CloudFormation Resource (AWS::CloudFront::StreamingDistribution.TrustedSigners)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-streamingdistribution-trustedsigners.html
type AWSCloudFrontStreamingDistribution_TrustedSigners struct {

	// AwsAccountNumbers AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-streamingdistribution-trustedsigners.html#cfn-cloudfront-streamingdistribution-trustedsigners-awsaccountnumbers
	AwsAccountNumbers []string `json:"AwsAccountNumbers,omitempty"`

	// Enabled AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-streamingdistribution-trustedsigners.html#cfn-cloudfront-streamingdistribution-trustedsigners-enabled
	Enabled bool `json:"Enabled,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCloudFrontStreamingDistribution_TrustedSigners) AWSCloudFormationType() string {
	return "AWS::CloudFront::StreamingDistribution.TrustedSigners"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCloudFrontStreamingDistribution_TrustedSigners) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
