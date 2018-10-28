package cloudformation

// AWSEC2SpotFleet_SpotFleetMonitoring AWS CloudFormation Resource (AWS::EC2::SpotFleet.SpotFleetMonitoring)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-spotfleet-spotfleetrequestconfigdata-launchspecifications-monitoring.html
type AWSEC2SpotFleet_SpotFleetMonitoring struct {

	// Enabled AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-spotfleet-spotfleetrequestconfigdata-launchspecifications-monitoring.html#cfn-ec2-spotfleet-spotfleetmonitoring-enabled
	Enabled bool `json:"Enabled,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEC2SpotFleet_SpotFleetMonitoring) AWSCloudFormationType() string {
	return "AWS::EC2::SpotFleet.SpotFleetMonitoring"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEC2SpotFleet_SpotFleetMonitoring) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
