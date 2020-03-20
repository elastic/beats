package pinpoint

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Campaign_EventDimensions AWS CloudFormation Resource (AWS::Pinpoint::Campaign.EventDimensions)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpoint-campaign-eventdimensions.html
type Campaign_EventDimensions struct {

	// Attributes AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpoint-campaign-eventdimensions.html#cfn-pinpoint-campaign-eventdimensions-attributes
	Attributes interface{} `json:"Attributes,omitempty"`

	// EventType AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpoint-campaign-eventdimensions.html#cfn-pinpoint-campaign-eventdimensions-eventtype
	EventType *Campaign_SetDimension `json:"EventType,omitempty"`

	// Metrics AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpoint-campaign-eventdimensions.html#cfn-pinpoint-campaign-eventdimensions-metrics
	Metrics interface{} `json:"Metrics,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Campaign_EventDimensions) AWSCloudFormationType() string {
	return "AWS::Pinpoint::Campaign.EventDimensions"
}
