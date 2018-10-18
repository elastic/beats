package cloudformation

// AWSElasticLoadBalancingLoadBalancer_ConnectionSettings AWS CloudFormation Resource (AWS::ElasticLoadBalancing::LoadBalancer.ConnectionSettings)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb-connectionsettings.html
type AWSElasticLoadBalancingLoadBalancer_ConnectionSettings struct {

	// IdleTimeout AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-elb-connectionsettings.html#cfn-elb-connectionsettings-idletimeout
	IdleTimeout int `json:"IdleTimeout,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSElasticLoadBalancingLoadBalancer_ConnectionSettings) AWSCloudFormationType() string {
	return "AWS::ElasticLoadBalancing::LoadBalancer.ConnectionSettings"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSElasticLoadBalancingLoadBalancer_ConnectionSettings) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
