package cloudformation

// AWSKinesisFirehoseDeliveryStream_KinesisStreamSourceConfiguration AWS CloudFormation Resource (AWS::KinesisFirehose::DeliveryStream.KinesisStreamSourceConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-kinesisstreamsourceconfiguration.html
type AWSKinesisFirehoseDeliveryStream_KinesisStreamSourceConfiguration struct {

	// KinesisStreamARN AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-kinesisstreamsourceconfiguration.html#cfn-kinesisfirehose-deliverystream-kinesisstreamsourceconfiguration-kinesisstreamarn
	KinesisStreamARN string `json:"KinesisStreamARN,omitempty"`

	// RoleARN AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-kinesisstreamsourceconfiguration.html#cfn-kinesisfirehose-deliverystream-kinesisstreamsourceconfiguration-rolearn
	RoleARN string `json:"RoleARN,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSKinesisFirehoseDeliveryStream_KinesisStreamSourceConfiguration) AWSCloudFormationType() string {
	return "AWS::KinesisFirehose::DeliveryStream.KinesisStreamSourceConfiguration"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSKinesisFirehoseDeliveryStream_KinesisStreamSourceConfiguration) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
