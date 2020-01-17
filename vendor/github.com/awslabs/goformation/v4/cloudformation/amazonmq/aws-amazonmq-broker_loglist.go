package amazonmq

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Broker_LogList AWS CloudFormation Resource (AWS::AmazonMQ::Broker.LogList)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-amazonmq-broker-loglist.html
type Broker_LogList struct {

	// Audit AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-amazonmq-broker-loglist.html#cfn-amazonmq-broker-loglist-audit
	Audit bool `json:"Audit,omitempty"`

	// General AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-amazonmq-broker-loglist.html#cfn-amazonmq-broker-loglist-general
	General bool `json:"General,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Broker_LogList) AWSCloudFormationType() string {
	return "AWS::AmazonMQ::Broker.LogList"
}
