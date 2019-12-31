package pinpoint

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Segment_Behavior AWS CloudFormation Resource (AWS::Pinpoint::Segment.Behavior)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpoint-segment-segmentdimensions-behavior.html
type Segment_Behavior struct {

	// Recency AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpoint-segment-segmentdimensions-behavior.html#cfn-pinpoint-segment-segmentdimensions-behavior-recency
	Recency *Segment_Recency `json:"Recency,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Segment_Behavior) AWSCloudFormationType() string {
	return "AWS::Pinpoint::Segment.Behavior"
}
