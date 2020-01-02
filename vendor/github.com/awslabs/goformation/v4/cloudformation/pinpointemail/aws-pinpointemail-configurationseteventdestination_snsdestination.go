package pinpointemail

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ConfigurationSetEventDestination_SnsDestination AWS CloudFormation Resource (AWS::PinpointEmail::ConfigurationSetEventDestination.SnsDestination)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpointemail-configurationseteventdestination-snsdestination.html
type ConfigurationSetEventDestination_SnsDestination struct {

	// TopicArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpointemail-configurationseteventdestination-snsdestination.html#cfn-pinpointemail-configurationseteventdestination-snsdestination-topicarn
	TopicArn string `json:"TopicArn,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ConfigurationSetEventDestination_SnsDestination) AWSCloudFormationType() string {
	return "AWS::PinpointEmail::ConfigurationSetEventDestination.SnsDestination"
}
