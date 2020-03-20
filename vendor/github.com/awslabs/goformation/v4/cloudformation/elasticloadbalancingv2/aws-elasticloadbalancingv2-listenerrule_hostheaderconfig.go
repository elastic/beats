package elasticloadbalancingv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ListenerRule_HostHeaderConfig AWS CloudFormation Resource (AWS::ElasticLoadBalancingV2::ListenerRule.HostHeaderConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-listenerrule-hostheaderconfig.html
type ListenerRule_HostHeaderConfig struct {

	// Values AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-listenerrule-hostheaderconfig.html#cfn-elasticloadbalancingv2-listenerrule-hostheaderconfig-values
	Values []string `json:"Values,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ListenerRule_HostHeaderConfig) AWSCloudFormationType() string {
	return "AWS::ElasticLoadBalancingV2::ListenerRule.HostHeaderConfig"
}
