package pinpointemail

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ConfigurationSet_ReputationOptions AWS CloudFormation Resource (AWS::PinpointEmail::ConfigurationSet.ReputationOptions)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpointemail-configurationset-reputationoptions.html
type ConfigurationSet_ReputationOptions struct {

	// ReputationMetricsEnabled AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpointemail-configurationset-reputationoptions.html#cfn-pinpointemail-configurationset-reputationoptions-reputationmetricsenabled
	ReputationMetricsEnabled bool `json:"ReputationMetricsEnabled,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ConfigurationSet_ReputationOptions) AWSCloudFormationType() string {
	return "AWS::PinpointEmail::ConfigurationSet.ReputationOptions"
}
