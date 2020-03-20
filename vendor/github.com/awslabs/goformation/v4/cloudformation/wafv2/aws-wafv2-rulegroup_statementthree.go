package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// RuleGroup_StatementThree AWS CloudFormation Resource (AWS::WAFv2::RuleGroup.StatementThree)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementthree.html
type RuleGroup_StatementThree struct {

	// ByteMatchStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementthree.html#cfn-wafv2-rulegroup-statementthree-bytematchstatement
	ByteMatchStatement *RuleGroup_ByteMatchStatement `json:"ByteMatchStatement,omitempty"`

	// GeoMatchStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementthree.html#cfn-wafv2-rulegroup-statementthree-geomatchstatement
	GeoMatchStatement *RuleGroup_GeoMatchStatement `json:"GeoMatchStatement,omitempty"`

	// IPSetReferenceStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementthree.html#cfn-wafv2-rulegroup-statementthree-ipsetreferencestatement
	IPSetReferenceStatement *RuleGroup_IPSetReferenceStatement `json:"IPSetReferenceStatement,omitempty"`

	// RegexPatternSetReferenceStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementthree.html#cfn-wafv2-rulegroup-statementthree-regexpatternsetreferencestatement
	RegexPatternSetReferenceStatement *RuleGroup_RegexPatternSetReferenceStatement `json:"RegexPatternSetReferenceStatement,omitempty"`

	// SizeConstraintStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementthree.html#cfn-wafv2-rulegroup-statementthree-sizeconstraintstatement
	SizeConstraintStatement *RuleGroup_SizeConstraintStatement `json:"SizeConstraintStatement,omitempty"`

	// SqliMatchStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementthree.html#cfn-wafv2-rulegroup-statementthree-sqlimatchstatement
	SqliMatchStatement *RuleGroup_SqliMatchStatement `json:"SqliMatchStatement,omitempty"`

	// XssMatchStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementthree.html#cfn-wafv2-rulegroup-statementthree-xssmatchstatement
	XssMatchStatement *RuleGroup_XssMatchStatement `json:"XssMatchStatement,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *RuleGroup_StatementThree) AWSCloudFormationType() string {
	return "AWS::WAFv2::RuleGroup.StatementThree"
}
