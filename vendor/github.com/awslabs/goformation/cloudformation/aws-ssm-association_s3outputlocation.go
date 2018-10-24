package cloudformation

// AWSSSMAssociation_S3OutputLocation AWS CloudFormation Resource (AWS::SSM::Association.S3OutputLocation)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-association-s3outputlocation.html
type AWSSSMAssociation_S3OutputLocation struct {

	// OutputS3BucketName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-association-s3outputlocation.html#cfn-ssm-association-s3outputlocation-outputs3bucketname
	OutputS3BucketName string `json:"OutputS3BucketName,omitempty"`

	// OutputS3KeyPrefix AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-association-s3outputlocation.html#cfn-ssm-association-s3outputlocation-outputs3keyprefix
	OutputS3KeyPrefix string `json:"OutputS3KeyPrefix,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSSMAssociation_S3OutputLocation) AWSCloudFormationType() string {
	return "AWS::SSM::Association.S3OutputLocation"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSSMAssociation_S3OutputLocation) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
