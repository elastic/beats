package cloudformation

// AWSElastiCacheReplicationGroup_NodeGroupConfiguration AWS CloudFormation Resource (AWS::ElastiCache::ReplicationGroup.NodeGroupConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticache-replicationgroup-nodegroupconfiguration.html
type AWSElastiCacheReplicationGroup_NodeGroupConfiguration struct {

	// PrimaryAvailabilityZone AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticache-replicationgroup-nodegroupconfiguration.html#cfn-elasticache-replicationgroup-nodegroupconfiguration-primaryavailabilityzone
	PrimaryAvailabilityZone string `json:"PrimaryAvailabilityZone,omitempty"`

	// ReplicaAvailabilityZones AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticache-replicationgroup-nodegroupconfiguration.html#cfn-elasticache-replicationgroup-nodegroupconfiguration-replicaavailabilityzones
	ReplicaAvailabilityZones []string `json:"ReplicaAvailabilityZones,omitempty"`

	// ReplicaCount AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticache-replicationgroup-nodegroupconfiguration.html#cfn-elasticache-replicationgroup-nodegroupconfiguration-replicacount
	ReplicaCount int `json:"ReplicaCount,omitempty"`

	// Slots AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticache-replicationgroup-nodegroupconfiguration.html#cfn-elasticache-replicationgroup-nodegroupconfiguration-slots
	Slots string `json:"Slots,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSElastiCacheReplicationGroup_NodeGroupConfiguration) AWSCloudFormationType() string {
	return "AWS::ElastiCache::ReplicationGroup.NodeGroupConfiguration"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSElastiCacheReplicationGroup_NodeGroupConfiguration) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
