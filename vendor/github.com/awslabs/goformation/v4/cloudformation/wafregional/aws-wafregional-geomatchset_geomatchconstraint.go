package wafregional

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// GeoMatchSet_GeoMatchConstraint AWS CloudFormation Resource (AWS::WAFRegional::GeoMatchSet.GeoMatchConstraint)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafregional-geomatchset-geomatchconstraint.html
type GeoMatchSet_GeoMatchConstraint struct {

	// Type AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafregional-geomatchset-geomatchconstraint.html#cfn-wafregional-geomatchset-geomatchconstraint-type
	Type string `json:"Type,omitempty"`

	// Value AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-wafregional-geomatchset-geomatchconstraint.html#cfn-wafregional-geomatchset-geomatchconstraint-value
	Value string `json:"Value,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *GeoMatchSet_GeoMatchConstraint) AWSCloudFormationType() string {
	return "AWS::WAFRegional::GeoMatchSet.GeoMatchConstraint"
}
