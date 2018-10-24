package cloudformation

// AWSServiceDiscoveryService_HealthCheckCustomConfig AWS CloudFormation Resource (AWS::ServiceDiscovery::Service.HealthCheckCustomConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicediscovery-service-healthcheckcustomconfig.html
type AWSServiceDiscoveryService_HealthCheckCustomConfig struct {

	// FailureThreshold AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicediscovery-service-healthcheckcustomconfig.html#cfn-servicediscovery-service-healthcheckcustomconfig-failurethreshold
	FailureThreshold float64 `json:"FailureThreshold,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSServiceDiscoveryService_HealthCheckCustomConfig) AWSCloudFormationType() string {
	return "AWS::ServiceDiscovery::Service.HealthCheckCustomConfig"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSServiceDiscoveryService_HealthCheckCustomConfig) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
