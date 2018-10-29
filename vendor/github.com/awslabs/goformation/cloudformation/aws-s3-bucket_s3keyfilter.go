package cloudformation

// AWSS3Bucket_S3KeyFilter AWS CloudFormation Resource (AWS::S3::Bucket.S3KeyFilter)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-notificationconfiguration-config-filter-s3key.html
type AWSS3Bucket_S3KeyFilter struct {

	// Rules AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-notificationconfiguration-config-filter-s3key.html#cfn-s3-bucket-notificationconfiguraiton-config-filter-s3key-rules
	Rules []AWSS3Bucket_FilterRule `json:"Rules,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSS3Bucket_S3KeyFilter) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.S3KeyFilter"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSS3Bucket_S3KeyFilter) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
