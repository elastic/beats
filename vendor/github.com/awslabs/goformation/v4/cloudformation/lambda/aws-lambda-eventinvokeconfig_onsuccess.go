package lambda

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// EventInvokeConfig_OnSuccess AWS CloudFormation Resource (AWS::Lambda::EventInvokeConfig.OnSuccess)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-eventinvokeconfig-destinationconfig-onsuccess.html
type EventInvokeConfig_OnSuccess struct {

	// Destination AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-eventinvokeconfig-destinationconfig-onsuccess.html#cfn-lambda-eventinvokeconfig-destinationconfig-onsuccess-destination
	Destination string `json:"Destination,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *EventInvokeConfig_OnSuccess) AWSCloudFormationType() string {
	return "AWS::Lambda::EventInvokeConfig.OnSuccess"
}
