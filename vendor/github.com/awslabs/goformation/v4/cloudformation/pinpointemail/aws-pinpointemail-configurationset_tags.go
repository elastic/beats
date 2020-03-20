package pinpointemail

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ConfigurationSet_Tags AWS CloudFormation Resource (AWS::PinpointEmail::ConfigurationSet.Tags)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpointemail-configurationset-tags.html
type ConfigurationSet_Tags struct {

	// Key AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpointemail-configurationset-tags.html#cfn-pinpointemail-configurationset-tags-key
	Key string `json:"Key,omitempty"`

	// Value AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpointemail-configurationset-tags.html#cfn-pinpointemail-configurationset-tags-value
	Value string `json:"Value,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ConfigurationSet_Tags) AWSCloudFormationType() string {
	return "AWS::PinpointEmail::ConfigurationSet.Tags"
}
