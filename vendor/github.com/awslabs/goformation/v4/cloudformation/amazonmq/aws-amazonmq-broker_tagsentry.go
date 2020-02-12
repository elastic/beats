package amazonmq

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Broker_TagsEntry AWS CloudFormation Resource (AWS::AmazonMQ::Broker.TagsEntry)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-amazonmq-broker-tagsentry.html
type Broker_TagsEntry struct {

	// Key AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-amazonmq-broker-tagsentry.html#cfn-amazonmq-broker-tagsentry-key
	Key string `json:"Key,omitempty"`

	// Value AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-amazonmq-broker-tagsentry.html#cfn-amazonmq-broker-tagsentry-value
	Value string `json:"Value,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Broker_TagsEntry) AWSCloudFormationType() string {
	return "AWS::AmazonMQ::Broker.TagsEntry"
}
