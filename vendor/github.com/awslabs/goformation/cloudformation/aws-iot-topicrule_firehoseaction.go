package cloudformation

// AWSIoTTopicRule_FirehoseAction AWS CloudFormation Resource (AWS::IoT::TopicRule.FirehoseAction)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iot-topicrule-firehoseaction.html
type AWSIoTTopicRule_FirehoseAction struct {

	// DeliveryStreamName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iot-topicrule-firehoseaction.html#cfn-iot-topicrule-firehoseaction-deliverystreamname
	DeliveryStreamName string `json:"DeliveryStreamName,omitempty"`

	// RoleArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iot-topicrule-firehoseaction.html#cfn-iot-topicrule-firehoseaction-rolearn
	RoleArn string `json:"RoleArn,omitempty"`

	// Separator AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iot-topicrule-firehoseaction.html#cfn-iot-topicrule-firehoseaction-separator
	Separator string `json:"Separator,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSIoTTopicRule_FirehoseAction) AWSCloudFormationType() string {
	return "AWS::IoT::TopicRule.FirehoseAction"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSIoTTopicRule_FirehoseAction) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
