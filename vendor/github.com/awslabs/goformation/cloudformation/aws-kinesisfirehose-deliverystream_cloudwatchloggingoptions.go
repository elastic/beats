package cloudformation

// AWSKinesisFirehoseDeliveryStream_CloudWatchLoggingOptions AWS CloudFormation Resource (AWS::KinesisFirehose::DeliveryStream.CloudWatchLoggingOptions)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-cloudwatchloggingoptions.html
type AWSKinesisFirehoseDeliveryStream_CloudWatchLoggingOptions struct {

	// Enabled AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-cloudwatchloggingoptions.html#cfn-kinesisfirehose-deliverystream-cloudwatchloggingoptions-enabled
	Enabled bool `json:"Enabled,omitempty"`

	// LogGroupName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-cloudwatchloggingoptions.html#cfn-kinesisfirehose-deliverystream-cloudwatchloggingoptions-loggroupname
	LogGroupName string `json:"LogGroupName,omitempty"`

	// LogStreamName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-cloudwatchloggingoptions.html#cfn-kinesisfirehose-deliverystream-cloudwatchloggingoptions-logstreamname
	LogStreamName string `json:"LogStreamName,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSKinesisFirehoseDeliveryStream_CloudWatchLoggingOptions) AWSCloudFormationType() string {
	return "AWS::KinesisFirehose::DeliveryStream.CloudWatchLoggingOptions"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSKinesisFirehoseDeliveryStream_CloudWatchLoggingOptions) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
