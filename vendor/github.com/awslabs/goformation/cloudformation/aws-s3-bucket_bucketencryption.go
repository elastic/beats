package cloudformation

// AWSS3Bucket_BucketEncryption AWS CloudFormation Resource (AWS::S3::Bucket.BucketEncryption)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-bucketencryption.html
type AWSS3Bucket_BucketEncryption struct {

	// ServerSideEncryptionConfiguration AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-bucketencryption.html#cfn-s3-bucket-bucketencryption-serversideencryptionconfiguration
	ServerSideEncryptionConfiguration []AWSS3Bucket_ServerSideEncryptionRule `json:"ServerSideEncryptionConfiguration,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSS3Bucket_BucketEncryption) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.BucketEncryption"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSS3Bucket_BucketEncryption) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
