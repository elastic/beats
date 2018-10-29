package cloudformation

// AWSRoute53HostedZone_QueryLoggingConfig AWS CloudFormation Resource (AWS::Route53::HostedZone.QueryLoggingConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-route53-hostedzone-queryloggingconfig.html
type AWSRoute53HostedZone_QueryLoggingConfig struct {

	// CloudWatchLogsLogGroupArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-route53-hostedzone-queryloggingconfig.html#cfn-route53-hostedzone-queryloggingconfig-cloudwatchlogsloggrouparn
	CloudWatchLogsLogGroupArn string `json:"CloudWatchLogsLogGroupArn,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSRoute53HostedZone_QueryLoggingConfig) AWSCloudFormationType() string {
	return "AWS::Route53::HostedZone.QueryLoggingConfig"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSRoute53HostedZone_QueryLoggingConfig) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
