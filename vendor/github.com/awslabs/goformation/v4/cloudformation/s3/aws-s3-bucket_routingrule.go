package s3

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Bucket_RoutingRule AWS CloudFormation Resource (AWS::S3::Bucket.RoutingRule)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-websiteconfiguration-routingrules.html
type Bucket_RoutingRule struct {

	// RedirectRule AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-websiteconfiguration-routingrules.html#cfn-s3-websiteconfiguration-routingrules-redirectrule
	RedirectRule *Bucket_RedirectRule `json:"RedirectRule,omitempty"`

	// RoutingRuleCondition AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-websiteconfiguration-routingrules.html#cfn-s3-websiteconfiguration-routingrules-routingrulecondition
	RoutingRuleCondition *Bucket_RoutingRuleCondition `json:"RoutingRuleCondition,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Bucket_RoutingRule) AWSCloudFormationType() string {
	return "AWS::S3::Bucket.RoutingRule"
}
