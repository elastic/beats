package elasticloadbalancingv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Listener_Action AWS CloudFormation Resource (AWS::ElasticLoadBalancingV2::Listener.Action)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-listener-defaultactions.html
type Listener_Action struct {

	// AuthenticateCognitoConfig AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-listener-defaultactions.html#cfn-elasticloadbalancingv2-listener-action-authenticatecognitoconfig
	AuthenticateCognitoConfig *Listener_AuthenticateCognitoConfig `json:"AuthenticateCognitoConfig,omitempty"`

	// AuthenticateOidcConfig AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-listener-defaultactions.html#cfn-elasticloadbalancingv2-listener-action-authenticateoidcconfig
	AuthenticateOidcConfig *Listener_AuthenticateOidcConfig `json:"AuthenticateOidcConfig,omitempty"`

	// FixedResponseConfig AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-listener-defaultactions.html#cfn-elasticloadbalancingv2-listener-action-fixedresponseconfig
	FixedResponseConfig *Listener_FixedResponseConfig `json:"FixedResponseConfig,omitempty"`

	// Order AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-listener-defaultactions.html#cfn-elasticloadbalancingv2-listener-action-order
	Order int `json:"Order,omitempty"`

	// RedirectConfig AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-listener-defaultactions.html#cfn-elasticloadbalancingv2-listener-action-redirectconfig
	RedirectConfig *Listener_RedirectConfig `json:"RedirectConfig,omitempty"`

	// TargetGroupArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-listener-defaultactions.html#cfn-elasticloadbalancingv2-listener-defaultactions-targetgrouparn
	TargetGroupArn string `json:"TargetGroupArn,omitempty"`

	// Type AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-listener-defaultactions.html#cfn-elasticloadbalancingv2-listener-defaultactions-type
	Type string `json:"Type,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Listener_Action) AWSCloudFormationType() string {
	return "AWS::ElasticLoadBalancingV2::Listener.Action"
}
