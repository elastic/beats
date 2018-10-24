package cloudformation

// AWSEMRCluster_PlacementType AWS CloudFormation Resource (AWS::EMR::Cluster.PlacementType)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-placementtype.html
type AWSEMRCluster_PlacementType struct {

	// AvailabilityZone AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-placementtype.html#cfn-elasticmapreduce-cluster-placementtype-availabilityzone
	AvailabilityZone string `json:"AvailabilityZone,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEMRCluster_PlacementType) AWSCloudFormationType() string {
	return "AWS::EMR::Cluster.PlacementType"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEMRCluster_PlacementType) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
