package cloudformation

// AWSS3Bucket_NotificationFilter AWS CloudFormation Resource (AWS::S3::Bucket.NotificationFilter)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-notificationconfiguration-config-filter.html
type AWSS3Bucket_NotificationFilter struct {

	// S3Key AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-notificationconfiguration-config-filter.html#cfn-s3-bucket-notificationconfiguraiton-config-filter-s3key
	S3Key *AWSS3Bucket_S3KeyFilter `json:"S3Key,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSS3Bucket_NotificationFilter) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.NotificationFilter"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSS3Bucket_NotificationFilter) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
