package managedblockchain

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Member_NetworkFrameworkConfiguration AWS CloudFormation Resource (AWS::ManagedBlockchain::Member.NetworkFrameworkConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-managedblockchain-member-networkframeworkconfiguration.html
type Member_NetworkFrameworkConfiguration struct {

	// NetworkFabricConfiguration AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-managedblockchain-member-networkframeworkconfiguration.html#cfn-managedblockchain-member-networkframeworkconfiguration-networkfabricconfiguration
	NetworkFabricConfiguration *Member_NetworkFabricConfiguration `json:"NetworkFabricConfiguration,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Member_NetworkFrameworkConfiguration) AWSCloudFormationType() string {
	return "AWS::ManagedBlockchain::Member.NetworkFrameworkConfiguration"
}
