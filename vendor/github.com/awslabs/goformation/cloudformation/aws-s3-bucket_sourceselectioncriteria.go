package cloudformation

// AWSS3Bucket_SourceSelectionCriteria AWS CloudFormation Resource (AWS::S3::Bucket.SourceSelectionCriteria)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-sourceselectioncriteria.html
type AWSS3Bucket_SourceSelectionCriteria struct {

	// SseKmsEncryptedObjects AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-sourceselectioncriteria.html#cfn-s3-bucket-sourceselectioncriteria-ssekmsencryptedobjects
	SseKmsEncryptedObjects *AWSS3Bucket_SseKmsEncryptedObjects `json:"SseKmsEncryptedObjects,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSS3Bucket_SourceSelectionCriteria) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.SourceSelectionCriteria"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSS3Bucket_SourceSelectionCriteria) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
