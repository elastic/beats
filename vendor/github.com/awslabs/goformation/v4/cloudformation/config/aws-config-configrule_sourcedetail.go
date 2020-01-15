package config

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ConfigRule_SourceDetail AWS CloudFormation Resource (AWS::Config::ConfigRule.SourceDetail)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-config-configrule-source-sourcedetails.html
type ConfigRule_SourceDetail struct {

	// EventSource AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-config-configrule-source-sourcedetails.html#cfn-config-configrule-source-sourcedetail-eventsource
	EventSource string `json:"EventSource,omitempty"`

	// MaximumExecutionFrequency AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-config-configrule-source-sourcedetails.html#cfn-config-configrule-sourcedetail-maximumexecutionfrequency
	MaximumExecutionFrequency string `json:"MaximumExecutionFrequency,omitempty"`

	// MessageType AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-config-configrule-source-sourcedetails.html#cfn-config-configrule-source-sourcedetail-messagetype
	MessageType string `json:"MessageType,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ConfigRule_SourceDetail) AWSCloudFormationType() string {
	return "AWS::Config::ConfigRule.SourceDetail"
}
