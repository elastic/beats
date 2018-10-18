package cloudformation

// AWSS3Bucket_DataExport AWS CloudFormation Resource (AWS::S3::Bucket.DataExport)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-dataexport.html
type AWSS3Bucket_DataExport struct {

	// Destination AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-dataexport.html#cfn-s3-bucket-dataexport-destination
	Destination *AWSS3Bucket_Destination `json:"Destination,omitempty"`

	// OutputSchemaVersion AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-bucket-dataexport.html#cfn-s3-bucket-dataexport-outputschemaversion
	OutputSchemaVersion string `json:"OutputSchemaVersion,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSS3Bucket_DataExport) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.DataExport"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSS3Bucket_DataExport) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
