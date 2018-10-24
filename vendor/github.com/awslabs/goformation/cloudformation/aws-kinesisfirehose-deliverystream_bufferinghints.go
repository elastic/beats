package cloudformation

// AWSKinesisFirehoseDeliveryStream_BufferingHints AWS CloudFormation Resource (AWS::KinesisFirehose::DeliveryStream.BufferingHints)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-bufferinghints.html
type AWSKinesisFirehoseDeliveryStream_BufferingHints struct {

	// IntervalInSeconds AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-bufferinghints.html#cfn-kinesisfirehose-deliverystream-bufferinghints-intervalinseconds
	IntervalInSeconds int `json:"IntervalInSeconds,omitempty"`

	// SizeInMBs AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-bufferinghints.html#cfn-kinesisfirehose-deliverystream-bufferinghints-sizeinmbs
	SizeInMBs int `json:"SizeInMBs,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSKinesisFirehoseDeliveryStream_BufferingHints) AWSCloudFormationType() string {
	return "AWS::KinesisFirehose::DeliveryStream.BufferingHints"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSKinesisFirehoseDeliveryStream_BufferingHints) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
