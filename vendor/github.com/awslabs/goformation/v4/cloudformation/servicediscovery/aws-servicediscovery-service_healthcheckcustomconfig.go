package servicediscovery

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Service_HealthCheckCustomConfig AWS CloudFormation Resource (AWS::ServiceDiscovery::Service.HealthCheckCustomConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicediscovery-service-healthcheckcustomconfig.html
type Service_HealthCheckCustomConfig struct {

	// FailureThreshold AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-servicediscovery-service-healthcheckcustomconfig.html#cfn-servicediscovery-service-healthcheckcustomconfig-failurethreshold
	FailureThreshold float64 `json:"FailureThreshold,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Service_HealthCheckCustomConfig) AWSCloudFormationType() string {
	return "AWS::ServiceDiscovery::Service.HealthCheckCustomConfig"
}
