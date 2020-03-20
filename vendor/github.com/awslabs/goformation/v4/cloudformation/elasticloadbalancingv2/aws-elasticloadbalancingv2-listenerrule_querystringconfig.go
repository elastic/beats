package elasticloadbalancingv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ListenerRule_QueryStringConfig AWS CloudFormation Resource (AWS::ElasticLoadBalancingV2::ListenerRule.QueryStringConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-listenerrule-querystringconfig.html
type ListenerRule_QueryStringConfig struct {

	// Values AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-listenerrule-querystringconfig.html#cfn-elasticloadbalancingv2-listenerrule-querystringconfig-values
	Values []ListenerRule_QueryStringKeyValue `json:"Values,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ListenerRule_QueryStringConfig) AWSCloudFormationType() string {
	return "AWS::ElasticLoadBalancingV2::ListenerRule.QueryStringConfig"
}
