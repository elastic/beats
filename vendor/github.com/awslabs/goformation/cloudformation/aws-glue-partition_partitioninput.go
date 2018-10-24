package cloudformation

// AWSGluePartition_PartitionInput AWS CloudFormation Resource (AWS::Glue::Partition.PartitionInput)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-partition-partitioninput.html
type AWSGluePartition_PartitionInput struct {

	// Parameters AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-partition-partitioninput.html#cfn-glue-partition-partitioninput-parameters
	Parameters interface{} `json:"Parameters,omitempty"`

	// StorageDescriptor AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-partition-partitioninput.html#cfn-glue-partition-partitioninput-storagedescriptor
	StorageDescriptor *AWSGluePartition_StorageDescriptor `json:"StorageDescriptor,omitempty"`

	// Values AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-partition-partitioninput.html#cfn-glue-partition-partitioninput-values
	Values []string `json:"Values,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSGluePartition_PartitionInput) AWSCloudFormationType() string {
	return "AWS::Glue::Partition.PartitionInput"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSGluePartition_PartitionInput) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
