package cloudformation

// AWSSESConfigurationSetEventDestination_EventDestination AWS CloudFormation Resource (AWS::SES::ConfigurationSetEventDestination.EventDestination)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-configurationseteventdestination-eventdestination.html
type AWSSESConfigurationSetEventDestination_EventDestination struct {

	// CloudWatchDestination AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-configurationseteventdestination-eventdestination.html#cfn-ses-configurationseteventdestination-eventdestination-cloudwatchdestination
	CloudWatchDestination *AWSSESConfigurationSetEventDestination_CloudWatchDestination `json:"CloudWatchDestination,omitempty"`

	// Enabled AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-configurationseteventdestination-eventdestination.html#cfn-ses-configurationseteventdestination-eventdestination-enabled
	Enabled bool `json:"Enabled,omitempty"`

	// KinesisFirehoseDestination AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-configurationseteventdestination-eventdestination.html#cfn-ses-configurationseteventdestination-eventdestination-kinesisfirehosedestination
	KinesisFirehoseDestination *AWSSESConfigurationSetEventDestination_KinesisFirehoseDestination `json:"KinesisFirehoseDestination,omitempty"`

	// MatchingEventTypes AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-configurationseteventdestination-eventdestination.html#cfn-ses-configurationseteventdestination-eventdestination-matchingeventtypes
	MatchingEventTypes []string `json:"MatchingEventTypes,omitempty"`

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-configurationseteventdestination-eventdestination.html#cfn-ses-configurationseteventdestination-eventdestination-name
	Name string `json:"Name,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSESConfigurationSetEventDestination_EventDestination) AWSCloudFormationType() string {
	return "AWS::SES::ConfigurationSetEventDestination.EventDestination"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSESConfigurationSetEventDestination_EventDestination) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
