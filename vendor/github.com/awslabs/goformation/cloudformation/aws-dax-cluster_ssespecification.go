package cloudformation

// AWSDAXCluster_SSESpecification AWS CloudFormation Resource (AWS::DAX::Cluster.SSESpecification)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dax-cluster-ssespecification.html
type AWSDAXCluster_SSESpecification struct {

	// SSEEnabled AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dax-cluster-ssespecification.html#cfn-dax-cluster-ssespecification-sseenabled
	SSEEnabled bool `json:"SSEEnabled,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSDAXCluster_SSESpecification) AWSCloudFormationType() string {
	return "AWS::DAX::Cluster.SSESpecification"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSDAXCluster_SSESpecification) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
