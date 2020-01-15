package managedblockchain

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Member_MemberFabricConfiguration AWS CloudFormation Resource (AWS::ManagedBlockchain::Member.MemberFabricConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-managedblockchain-member-memberfabricconfiguration.html
type Member_MemberFabricConfiguration struct {

	// AdminPassword AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-managedblockchain-member-memberfabricconfiguration.html#cfn-managedblockchain-member-memberfabricconfiguration-adminpassword
	AdminPassword string `json:"AdminPassword,omitempty"`

	// AdminUsername AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-managedblockchain-member-memberfabricconfiguration.html#cfn-managedblockchain-member-memberfabricconfiguration-adminusername
	AdminUsername string `json:"AdminUsername,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Member_MemberFabricConfiguration) AWSCloudFormationType() string {
	return "AWS::ManagedBlockchain::Member.MemberFabricConfiguration"
}
