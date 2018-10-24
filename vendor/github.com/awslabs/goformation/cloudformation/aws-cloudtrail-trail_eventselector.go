package cloudformation

// AWSCloudTrailTrail_EventSelector AWS CloudFormation Resource (AWS::CloudTrail::Trail.EventSelector)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudtrail-trail-eventselector.html
type AWSCloudTrailTrail_EventSelector struct {

	// DataResources AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudtrail-trail-eventselector.html#cfn-cloudtrail-trail-eventselector-dataresources
	DataResources []AWSCloudTrailTrail_DataResource `json:"DataResources,omitempty"`

	// IncludeManagementEvents AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudtrail-trail-eventselector.html#cfn-cloudtrail-trail-eventselector-includemanagementevents
	IncludeManagementEvents bool `json:"IncludeManagementEvents,omitempty"`

	// ReadWriteType AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudtrail-trail-eventselector.html#cfn-cloudtrail-trail-eventselector-readwritetype
	ReadWriteType string `json:"ReadWriteType,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCloudTrailTrail_EventSelector) AWSCloudFormationType() string {
	return "AWS::CloudTrail::Trail.EventSelector"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCloudTrailTrail_EventSelector) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
