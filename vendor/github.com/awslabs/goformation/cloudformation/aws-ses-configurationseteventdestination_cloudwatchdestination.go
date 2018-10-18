package cloudformation

// AWSSESConfigurationSetEventDestination_CloudWatchDestination AWS CloudFormation Resource (AWS::SES::ConfigurationSetEventDestination.CloudWatchDestination)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-configurationseteventdestination-cloudwatchdestination.html
type AWSSESConfigurationSetEventDestination_CloudWatchDestination struct {

	// DimensionConfigurations AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-configurationseteventdestination-cloudwatchdestination.html#cfn-ses-configurationseteventdestination-cloudwatchdestination-dimensionconfigurations
	DimensionConfigurations []AWSSESConfigurationSetEventDestination_DimensionConfiguration `json:"DimensionConfigurations,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSESConfigurationSetEventDestination_CloudWatchDestination) AWSCloudFormationType() string {
	return "AWS::SES::ConfigurationSetEventDestination.CloudWatchDestination"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSESConfigurationSetEventDestination_CloudWatchDestination) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
