package cloudformation

// AWSS3Bucket_ServerSideEncryptionByDefault AWS CloudFormation Resource (AWS::S3::Bucket.ServerSideEncryptionByDefault)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-serversideencryptionbydefault.html
type AWSS3Bucket_ServerSideEncryptionByDefault struct {

	// KMSMasterKeyID AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-serversideencryptionbydefault.html#cfn-s3-bucket-serversideencryptionbydefault-kmsmasterkeyid
	KMSMasterKeyID string `json:"KMSMasterKeyID,omitempty"`

	// SSEAlgorithm AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-serversideencryptionbydefault.html#cfn-s3-bucket-serversideencryptionbydefault-ssealgorithm
	SSEAlgorithm string `json:"SSEAlgorithm,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSS3Bucket_ServerSideEncryptionByDefault) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.ServerSideEncryptionByDefault"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSS3Bucket_ServerSideEncryptionByDefault) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
