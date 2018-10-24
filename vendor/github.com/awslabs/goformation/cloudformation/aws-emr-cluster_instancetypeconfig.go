package cloudformation

// AWSEMRCluster_InstanceTypeConfig AWS CloudFormation Resource (AWS::EMR::Cluster.InstanceTypeConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-instancetypeconfig.html
type AWSEMRCluster_InstanceTypeConfig struct {

	// BidPrice AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-instancetypeconfig.html#cfn-elasticmapreduce-cluster-instancetypeconfig-bidprice
	BidPrice string `json:"BidPrice,omitempty"`

	// BidPriceAsPercentageOfOnDemandPrice AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-instancetypeconfig.html#cfn-elasticmapreduce-cluster-instancetypeconfig-bidpriceaspercentageofondemandprice
	BidPriceAsPercentageOfOnDemandPrice float64 `json:"BidPriceAsPercentageOfOnDemandPrice,omitempty"`

	// Configurations AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-instancetypeconfig.html#cfn-elasticmapreduce-cluster-instancetypeconfig-configurations
	Configurations []AWSEMRCluster_Configuration `json:"Configurations,omitempty"`

	// EbsConfiguration AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-instancetypeconfig.html#cfn-elasticmapreduce-cluster-instancetypeconfig-ebsconfiguration
	EbsConfiguration *AWSEMRCluster_EbsConfiguration `json:"EbsConfiguration,omitempty"`

	// InstanceType AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-instancetypeconfig.html#cfn-elasticmapreduce-cluster-instancetypeconfig-instancetype
	InstanceType string `json:"InstanceType,omitempty"`

	// WeightedCapacity AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticmapreduce-cluster-instancetypeconfig.html#cfn-elasticmapreduce-cluster-instancetypeconfig-weightedcapacity
	WeightedCapacity int `json:"WeightedCapacity,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEMRCluster_InstanceTypeConfig) AWSCloudFormationType() string {
	return "AWS::EMR::Cluster.InstanceTypeConfig"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEMRCluster_InstanceTypeConfig) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
