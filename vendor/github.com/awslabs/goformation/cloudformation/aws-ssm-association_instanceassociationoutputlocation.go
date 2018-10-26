package cloudformation

// AWSSSMAssociation_InstanceAssociationOutputLocation AWS CloudFormation Resource (AWS::SSM::Association.InstanceAssociationOutputLocation)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-association-instanceassociationoutputlocation.html
type AWSSSMAssociation_InstanceAssociationOutputLocation struct {

	// S3Location AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-association-instanceassociationoutputlocation.html#cfn-ssm-association-instanceassociationoutputlocation-s3location
	S3Location *AWSSSMAssociation_S3OutputLocation `json:"S3Location,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSSMAssociation_InstanceAssociationOutputLocation) AWSCloudFormationType() string {
	return "AWS::SSM::Association.InstanceAssociationOutputLocation"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSSMAssociation_InstanceAssociationOutputLocation) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
