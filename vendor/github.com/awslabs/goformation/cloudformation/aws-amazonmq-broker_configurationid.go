package cloudformation

// AWSAmazonMQBroker_ConfigurationId AWS CloudFormation Resource (AWS::AmazonMQ::Broker.ConfigurationId)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-amazonmq-broker-configurationid.html
type AWSAmazonMQBroker_ConfigurationId struct {

	// Id AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-amazonmq-broker-configurationid.html#cfn-amazonmq-broker-configurationid-id
	Id string `json:"Id,omitempty"`

	// Revision AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-amazonmq-broker-configurationid.html#cfn-amazonmq-broker-configurationid-revision
	Revision int `json:"Revision,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSAmazonMQBroker_ConfigurationId) AWSCloudFormationType() string {
	return "AWS::AmazonMQ::Broker.ConfigurationId"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSAmazonMQBroker_ConfigurationId) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
