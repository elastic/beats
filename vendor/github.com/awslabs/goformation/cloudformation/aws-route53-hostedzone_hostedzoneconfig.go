package cloudformation

// AWSRoute53HostedZone_HostedZoneConfig AWS CloudFormation Resource (AWS::Route53::HostedZone.HostedZoneConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-route53-hostedzone-hostedzoneconfig.html
type AWSRoute53HostedZone_HostedZoneConfig struct {

	// Comment AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-route53-hostedzone-hostedzoneconfig.html#cfn-route53-hostedzone-hostedzoneconfig-comment
	Comment string `json:"Comment,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSRoute53HostedZone_HostedZoneConfig) AWSCloudFormationType() string {
	return "AWS::Route53::HostedZone.HostedZoneConfig"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSRoute53HostedZone_HostedZoneConfig) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
