package cloudformation

// AWSSESConfigurationSetEventDestination_KinesisFirehoseDestination AWS CloudFormation Resource (AWS::SES::ConfigurationSetEventDestination.KinesisFirehoseDestination)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-configurationseteventdestination-kinesisfirehosedestination.html
type AWSSESConfigurationSetEventDestination_KinesisFirehoseDestination struct {

	// DeliveryStreamARN AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-configurationseteventdestination-kinesisfirehosedestination.html#cfn-ses-configurationseteventdestination-kinesisfirehosedestination-deliverystreamarn
	DeliveryStreamARN string `json:"DeliveryStreamARN,omitempty"`

	// IAMRoleARN AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ses-configurationseteventdestination-kinesisfirehosedestination.html#cfn-ses-configurationseteventdestination-kinesisfirehosedestination-iamrolearn
	IAMRoleARN string `json:"IAMRoleARN,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSESConfigurationSetEventDestination_KinesisFirehoseDestination) AWSCloudFormationType() string {
	return "AWS::SES::ConfigurationSetEventDestination.KinesisFirehoseDestination"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSESConfigurationSetEventDestination_KinesisFirehoseDestination) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
