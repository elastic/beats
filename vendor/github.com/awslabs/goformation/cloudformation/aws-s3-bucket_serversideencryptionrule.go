package cloudformation

// AWSS3Bucket_ServerSideEncryptionRule AWS CloudFormation Resource (AWS::S3::Bucket.ServerSideEncryptionRule)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-serversideencryptionrule.html
type AWSS3Bucket_ServerSideEncryptionRule struct {

	// ServerSideEncryptionByDefault AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-serversideencryptionrule.html#cfn-s3-bucket-serversideencryptionrule-serversideencryptionbydefault
	ServerSideEncryptionByDefault *AWSS3Bucket_ServerSideEncryptionByDefault `json:"ServerSideEncryptionByDefault,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSS3Bucket_ServerSideEncryptionRule) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.ServerSideEncryptionRule"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSS3Bucket_ServerSideEncryptionRule) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
