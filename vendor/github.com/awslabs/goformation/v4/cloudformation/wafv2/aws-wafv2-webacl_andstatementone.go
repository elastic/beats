package wafv2

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// WebACL_AndStatementOne AWS CloudFormation Resource (AWS::WAFv2::WebACL.AndStatementOne)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-andstatementone.html
type WebACL_AndStatementOne struct {

	// Statements AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafv2-webacl-andstatementone.html#cfn-wafv2-webacl-andstatementone-statements
	Statements *WebACL_StatementTwos `json:"Statements,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *WebACL_AndStatementOne) AWSCloudFormationType() string {
	return "AWS::WAFv2::WebACL.AndStatementOne"
}
