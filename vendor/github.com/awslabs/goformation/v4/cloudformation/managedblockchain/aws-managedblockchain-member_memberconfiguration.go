package managedblockchain

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Member_MemberConfiguration AWS CloudFormation Resource (AWS::ManagedBlockchain::Member.MemberConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-managedblockchain-member-memberconfiguration.html
type Member_MemberConfiguration struct {

	// Description AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-managedblockchain-member-memberconfiguration.html#cfn-managedblockchain-member-memberconfiguration-description
	Description string `json:"Description,omitempty"`

	// MemberFrameworkConfiguration AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-managedblockchain-member-memberconfiguration.html#cfn-managedblockchain-member-memberconfiguration-memberframeworkconfiguration
	MemberFrameworkConfiguration *Member_MemberFrameworkConfiguration `json:"MemberFrameworkConfiguration,omitempty"`

	// Name AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-managedblockchain-member-memberconfiguration.html#cfn-managedblockchain-member-memberconfiguration-name
	Name string `json:"Name,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Member_MemberConfiguration) AWSCloudFormationType() string {
	return "AWS::ManagedBlockchain::Member.MemberConfiguration"
}
