package wafregional

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Rule_Predicate AWS CloudFormation Resource (AWS::WAFRegional::Rule.Predicate)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafregional-rule-predicate.html
type Rule_Predicate struct {

	// DataId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafregional-rule-predicate.html#cfn-wafregional-rule-predicate-dataid
	DataId string `json:"DataId,omitempty"`

	// Negated AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafregional-rule-predicate.html#cfn-wafregional-rule-predicate-negated
	Negated bool `json:"Negated"`

	// Type AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafregional-rule-predicate.html#cfn-wafregional-rule-predicate-type
	Type string `json:"Type,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Rule_Predicate) AWSCloudFormationType() string {
	return "AWS::WAFRegional::Rule.Predicate"
}
