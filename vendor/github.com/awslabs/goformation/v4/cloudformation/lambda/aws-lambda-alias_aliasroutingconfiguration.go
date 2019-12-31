package lambda

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Alias_AliasRoutingConfiguration AWS CloudFormation Resource (AWS::Lambda::Alias.AliasRoutingConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-alias-aliasroutingconfiguration.html
type Alias_AliasRoutingConfiguration struct {

	// AdditionalVersionWeights AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-alias-aliasroutingconfiguration.html#cfn-lambda-alias-aliasroutingconfiguration-additionalversionweights
	AdditionalVersionWeights []Alias_VersionWeight `json:"AdditionalVersionWeights,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Alias_AliasRoutingConfiguration) AWSCloudFormationType() string {
	return "AWS::Lambda::Alias.AliasRoutingConfiguration"
}
