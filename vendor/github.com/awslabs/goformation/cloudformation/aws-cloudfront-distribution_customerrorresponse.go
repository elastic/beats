package cloudformation

// AWSCloudFrontDistribution_CustomErrorResponse AWS CloudFormation Resource (AWS::CloudFront::Distribution.CustomErrorResponse)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-customerrorresponse.html
type AWSCloudFrontDistribution_CustomErrorResponse struct {

	// ErrorCachingMinTTL AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-customerrorresponse.html#cfn-cloudfront-distribution-customerrorresponse-errorcachingminttl
	ErrorCachingMinTTL float64 `json:"ErrorCachingMinTTL,omitempty"`

	// ErrorCode AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-customerrorresponse.html#cfn-cloudfront-distribution-customerrorresponse-errorcode
	ErrorCode int `json:"ErrorCode,omitempty"`

	// ResponseCode AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-customerrorresponse.html#cfn-cloudfront-distribution-customerrorresponse-responsecode
	ResponseCode int `json:"ResponseCode,omitempty"`

	// ResponsePagePath AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-distribution-customerrorresponse.html#cfn-cloudfront-distribution-customerrorresponse-responsepagepath
	ResponsePagePath string `json:"ResponsePagePath,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCloudFrontDistribution_CustomErrorResponse) AWSCloudFormationType() string {
	return "AWS::CloudFront::Distribution.CustomErrorResponse"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCloudFrontDistribution_CustomErrorResponse) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
