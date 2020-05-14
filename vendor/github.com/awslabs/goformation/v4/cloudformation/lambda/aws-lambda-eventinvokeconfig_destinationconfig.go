package lambda

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// EventInvokeConfig_DestinationConfig AWS CloudFormation Resource (AWS::Lambda::EventInvokeConfig.DestinationConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-eventinvokeconfig-destinationconfig.html
type EventInvokeConfig_DestinationConfig struct {

	// OnFailure AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-eventinvokeconfig-destinationconfig.html#cfn-lambda-eventinvokeconfig-destinationconfig-onfailure
	OnFailure *EventInvokeConfig_OnFailure `json:"OnFailure,omitempty"`

	// OnSuccess AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-eventinvokeconfig-destinationconfig.html#cfn-lambda-eventinvokeconfig-destinationconfig-onsuccess
	OnSuccess *EventInvokeConfig_OnSuccess `json:"OnSuccess,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *EventInvokeConfig_DestinationConfig) AWSCloudFormationType() string {
	return "AWS::Lambda::EventInvokeConfig.DestinationConfig"
}
