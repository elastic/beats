package cloudformation

// AWSS3Bucket_AccessControlTranslation AWS CloudFormation Resource (AWS::S3::Bucket.AccessControlTranslation)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-accesscontroltranslation.html
type AWSS3Bucket_AccessControlTranslation struct {

	// Owner AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-accesscontroltranslation.html#cfn-s3-bucket-accesscontroltranslation-owner
	Owner string `json:"Owner,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSS3Bucket_AccessControlTranslation) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.AccessControlTranslation"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSS3Bucket_AccessControlTranslation) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
