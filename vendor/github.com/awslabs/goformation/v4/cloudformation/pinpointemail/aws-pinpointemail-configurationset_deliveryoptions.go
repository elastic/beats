package pinpointemail

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ConfigurationSet_DeliveryOptions AWS CloudFormation Resource (AWS::PinpointEmail::ConfigurationSet.DeliveryOptions)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpointemail-configurationset-deliveryoptions.html
type ConfigurationSet_DeliveryOptions struct {

	// SendingPoolName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpointemail-configurationset-deliveryoptions.html#cfn-pinpointemail-configurationset-deliveryoptions-sendingpoolname
	SendingPoolName string `json:"SendingPoolName,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ConfigurationSet_DeliveryOptions) AWSCloudFormationType() string {
	return "AWS::PinpointEmail::ConfigurationSet.DeliveryOptions"
}
