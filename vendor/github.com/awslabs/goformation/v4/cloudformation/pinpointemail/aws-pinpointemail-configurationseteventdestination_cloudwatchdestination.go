package pinpointemail

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ConfigurationSetEventDestination_CloudWatchDestination AWS CloudFormation Resource (AWS::PinpointEmail::ConfigurationSetEventDestination.CloudWatchDestination)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpointemail-configurationseteventdestination-cloudwatchdestination.html
type ConfigurationSetEventDestination_CloudWatchDestination struct {

	// DimensionConfigurations AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpointemail-configurationseteventdestination-cloudwatchdestination.html#cfn-pinpointemail-configurationseteventdestination-cloudwatchdestination-dimensionconfigurations
	DimensionConfigurations []ConfigurationSetEventDestination_DimensionConfiguration `json:"DimensionConfigurations,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ConfigurationSetEventDestination_CloudWatchDestination) AWSCloudFormationType() string {
	return "AWS::PinpointEmail::ConfigurationSetEventDestination.CloudWatchDestination"
}
