package cloudformation

// AWSEMRCluster_KerberosAttributes AWS CloudFormation Resource (AWS::EMR::Cluster.KerberosAttributes)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-kerberosattributes.html
type AWSEMRCluster_KerberosAttributes struct {

	// ADDomainJoinPassword AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-kerberosattributes.html#cfn-elasticmapreduce-cluster-kerberosattributes-addomainjoinpassword
	ADDomainJoinPassword string `json:"ADDomainJoinPassword,omitempty"`

	// ADDomainJoinUser AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-kerberosattributes.html#cfn-elasticmapreduce-cluster-kerberosattributes-addomainjoinuser
	ADDomainJoinUser string `json:"ADDomainJoinUser,omitempty"`

	// CrossRealmTrustPrincipalPassword AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-kerberosattributes.html#cfn-elasticmapreduce-cluster-kerberosattributes-crossrealmtrustprincipalpassword
	CrossRealmTrustPrincipalPassword string `json:"CrossRealmTrustPrincipalPassword,omitempty"`

	// KdcAdminPassword AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-kerberosattributes.html#cfn-elasticmapreduce-cluster-kerberosattributes-kdcadminpassword
	KdcAdminPassword string `json:"KdcAdminPassword,omitempty"`

	// Realm AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-kerberosattributes.html#cfn-elasticmapreduce-cluster-kerberosattributes-realm
	Realm string `json:"Realm,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEMRCluster_KerberosAttributes) AWSCloudFormationType() string {
	return "AWS::EMR::Cluster.KerberosAttributes"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEMRCluster_KerberosAttributes) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
