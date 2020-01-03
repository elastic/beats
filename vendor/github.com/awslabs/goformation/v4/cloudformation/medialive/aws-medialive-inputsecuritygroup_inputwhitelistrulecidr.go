package medialive

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// InputSecurityGroup_InputWhitelistRuleCidr AWS CloudFormation Resource (AWS::MediaLive::InputSecurityGroup.InputWhitelistRuleCidr)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-inputsecuritygroup-inputwhitelistrulecidr.html
type InputSecurityGroup_InputWhitelistRuleCidr struct {

	// Cidr AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-medialive-inputsecuritygroup-inputwhitelistrulecidr.html#cfn-medialive-inputsecuritygroup-inputwhitelistrulecidr-cidr
	Cidr string `json:"Cidr,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *InputSecurityGroup_InputWhitelistRuleCidr) AWSCloudFormationType() string {
	return "AWS::MediaLive::InputSecurityGroup.InputWhitelistRuleCidr"
}
