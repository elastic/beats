package cloudformation

// AWSKinesisFirehoseDeliveryStream_ProcessorParameter AWS CloudFormation Resource (AWS::KinesisFirehose::DeliveryStream.ProcessorParameter)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-processorparameter.html
type AWSKinesisFirehoseDeliveryStream_ProcessorParameter struct {

	// ParameterName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-processorparameter.html#cfn-kinesisfirehose-deliverystream-processorparameter-parametername
	ParameterName string `json:"ParameterName,omitempty"`

	// ParameterValue AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-processorparameter.html#cfn-kinesisfirehose-deliverystream-processorparameter-parametervalue
	ParameterValue string `json:"ParameterValue,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSKinesisFirehoseDeliveryStream_ProcessorParameter) AWSCloudFormationType() string {
	return "AWS::KinesisFirehose::DeliveryStream.ProcessorParameter"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSKinesisFirehoseDeliveryStream_ProcessorParameter) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
