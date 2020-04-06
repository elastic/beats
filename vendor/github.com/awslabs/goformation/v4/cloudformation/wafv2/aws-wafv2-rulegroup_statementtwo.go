package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// RuleGroup_StatementTwo AWS CloudFormation Resource (AWS::WAFv2::RuleGroup.StatementTwo)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementtwo.html
type RuleGroup_StatementTwo struct {

	// AndStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementtwo.html#cfn-wafv2-rulegroup-statementtwo-andstatement
	AndStatement *RuleGroup_AndStatementTwo `json:"AndStatement,omitempty"`

	// ByteMatchStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementtwo.html#cfn-wafv2-rulegroup-statementtwo-bytematchstatement
	ByteMatchStatement *RuleGroup_ByteMatchStatement `json:"ByteMatchStatement,omitempty"`

	// GeoMatchStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementtwo.html#cfn-wafv2-rulegroup-statementtwo-geomatchstatement
	GeoMatchStatement *RuleGroup_GeoMatchStatement `json:"GeoMatchStatement,omitempty"`

	// IPSetReferenceStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementtwo.html#cfn-wafv2-rulegroup-statementtwo-ipsetreferencestatement
	IPSetReferenceStatement *RuleGroup_IPSetReferenceStatement `json:"IPSetReferenceStatement,omitempty"`

	// NotStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementtwo.html#cfn-wafv2-rulegroup-statementtwo-notstatement
	NotStatement *RuleGroup_NotStatementTwo `json:"NotStatement,omitempty"`

	// OrStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementtwo.html#cfn-wafv2-rulegroup-statementtwo-orstatement
	OrStatement *RuleGroup_OrStatementTwo `json:"OrStatement,omitempty"`

	// RateBasedStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementtwo.html#cfn-wafv2-rulegroup-statementtwo-ratebasedstatement
	RateBasedStatement *RuleGroup_RateBasedStatementTwo `json:"RateBasedStatement,omitempty"`

	// RegexPatternSetReferenceStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementtwo.html#cfn-wafv2-rulegroup-statementtwo-regexpatternsetreferencestatement
	RegexPatternSetReferenceStatement *RuleGroup_RegexPatternSetReferenceStatement `json:"RegexPatternSetReferenceStatement,omitempty"`

	// SizeConstraintStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementtwo.html#cfn-wafv2-rulegroup-statementtwo-sizeconstraintstatement
	SizeConstraintStatement *RuleGroup_SizeConstraintStatement `json:"SizeConstraintStatement,omitempty"`

	// SqliMatchStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementtwo.html#cfn-wafv2-rulegroup-statementtwo-sqlimatchstatement
	SqliMatchStatement *RuleGroup_SqliMatchStatement `json:"SqliMatchStatement,omitempty"`

	// XssMatchStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-rulegroup-statementtwo.html#cfn-wafv2-rulegroup-statementtwo-xssmatchstatement
	XssMatchStatement *RuleGroup_XssMatchStatement `json:"XssMatchStatement,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *RuleGroup_StatementTwo) AWSCloudFormationType() string {
	return "AWS::WAFv2::RuleGroup.StatementTwo"
}
