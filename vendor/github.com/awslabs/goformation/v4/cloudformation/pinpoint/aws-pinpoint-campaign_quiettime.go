package pinpoint

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Campaign_QuietTime AWS CloudFormation Resource (AWS::Pinpoint::Campaign.QuietTime)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpoint-campaign-schedule-quiettime.html
type Campaign_QuietTime struct {

	// End AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpoint-campaign-schedule-quiettime.html#cfn-pinpoint-campaign-schedule-quiettime-end
	End string `json:"End,omitempty"`

	// Start AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpoint-campaign-schedule-quiettime.html#cfn-pinpoint-campaign-schedule-quiettime-start
	Start string `json:"Start,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Campaign_QuietTime) AWSCloudFormationType() string {
	return "AWS::Pinpoint::Campaign.QuietTime"
}
