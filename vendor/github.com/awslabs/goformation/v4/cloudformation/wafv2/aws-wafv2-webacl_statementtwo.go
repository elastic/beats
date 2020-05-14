package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// WebACL_StatementTwo AWS CloudFormation Resource (AWS::WAFv2::WebACL.StatementTwo)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-statementtwo.html
type WebACL_StatementTwo struct {

	// AndStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-statementtwo.html#cfn-wafv2-webacl-statementtwo-andstatement
	AndStatement *WebACL_AndStatementTwo `json:"AndStatement,omitempty"`

	// ByteMatchStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-statementtwo.html#cfn-wafv2-webacl-statementtwo-bytematchstatement
	ByteMatchStatement *WebACL_ByteMatchStatement `json:"ByteMatchStatement,omitempty"`

	// GeoMatchStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-statementtwo.html#cfn-wafv2-webacl-statementtwo-geomatchstatement
	GeoMatchStatement *WebACL_GeoMatchStatement `json:"GeoMatchStatement,omitempty"`

	// IPSetReferenceStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-statementtwo.html#cfn-wafv2-webacl-statementtwo-ipsetreferencestatement
	IPSetReferenceStatement *WebACL_IPSetReferenceStatement `json:"IPSetReferenceStatement,omitempty"`

	// ManagedRuleGroupStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-statementtwo.html#cfn-wafv2-webacl-statementtwo-managedrulegroupstatement
	ManagedRuleGroupStatement *WebACL_ManagedRuleGroupStatement `json:"ManagedRuleGroupStatement,omitempty"`

	// NotStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-statementtwo.html#cfn-wafv2-webacl-statementtwo-notstatement
	NotStatement *WebACL_NotStatementTwo `json:"NotStatement,omitempty"`

	// OrStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-statementtwo.html#cfn-wafv2-webacl-statementtwo-orstatement
	OrStatement *WebACL_OrStatementTwo `json:"OrStatement,omitempty"`

	// RateBasedStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-statementtwo.html#cfn-wafv2-webacl-statementtwo-ratebasedstatement
	RateBasedStatement *WebACL_RateBasedStatementTwo `json:"RateBasedStatement,omitempty"`

	// RegexPatternSetReferenceStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-statementtwo.html#cfn-wafv2-webacl-statementtwo-regexpatternsetreferencestatement
	RegexPatternSetReferenceStatement *WebACL_RegexPatternSetReferenceStatement `json:"RegexPatternSetReferenceStatement,omitempty"`

	// RuleGroupReferenceStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-statementtwo.html#cfn-wafv2-webacl-statementtwo-rulegroupreferencestatement
	RuleGroupReferenceStatement *WebACL_RuleGroupReferenceStatement `json:"RuleGroupReferenceStatement,omitempty"`

	// SizeConstraintStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-statementtwo.html#cfn-wafv2-webacl-statementtwo-sizeconstraintstatement
	SizeConstraintStatement *WebACL_SizeConstraintStatement `json:"SizeConstraintStatement,omitempty"`

	// SqliMatchStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-statementtwo.html#cfn-wafv2-webacl-statementtwo-sqlimatchstatement
	SqliMatchStatement *WebACL_SqliMatchStatement `json:"SqliMatchStatement,omitempty"`

	// XssMatchStatement AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-statementtwo.html#cfn-wafv2-webacl-statementtwo-xssmatchstatement
	XssMatchStatement *WebACL_XssMatchStatement `json:"XssMatchStatement,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *WebACL_StatementTwo) AWSCloudFormationType() string {
	return "AWS::WAFv2::WebACL.StatementTwo"
}
