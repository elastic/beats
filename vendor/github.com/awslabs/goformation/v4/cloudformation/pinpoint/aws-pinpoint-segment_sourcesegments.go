package pinpoint

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Segment_SourceSegments AWS CloudFormation Resource (AWS::Pinpoint::Segment.SourceSegments)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpoint-segment-segmentgroups-groups-sourcesegments.html
type Segment_SourceSegments struct {

	// Id AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpoint-segment-segmentgroups-groups-sourcesegments.html#cfn-pinpoint-segment-segmentgroups-groups-sourcesegments-id
	Id string `json:"Id,omitempty"`

	// Version AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-pinpoint-segment-segmentgroups-groups-sourcesegments.html#cfn-pinpoint-segment-segmentgroups-groups-sourcesegments-version
	Version int `json:"Version,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Segment_SourceSegments) AWSCloudFormationType() string {
	return "AWS::Pinpoint::Segment.SourceSegments"
}
