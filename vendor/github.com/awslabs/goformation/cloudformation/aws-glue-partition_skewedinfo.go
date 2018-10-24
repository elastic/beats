package cloudformation

// AWSGluePartition_SkewedInfo AWS CloudFormation Resource (AWS::Glue::Partition.SkewedInfo)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-partition-skewedinfo.html
type AWSGluePartition_SkewedInfo struct {

	// SkewedColumnNames AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-partition-skewedinfo.html#cfn-glue-partition-skewedinfo-skewedcolumnnames
	SkewedColumnNames []string `json:"SkewedColumnNames,omitempty"`

	// SkewedColumnValueLocationMaps AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-partition-skewedinfo.html#cfn-glue-partition-skewedinfo-skewedcolumnvaluelocationmaps
	SkewedColumnValueLocationMaps interface{} `json:"SkewedColumnValueLocationMaps,omitempty"`

	// SkewedColumnValues AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-partition-skewedinfo.html#cfn-glue-partition-skewedinfo-skewedcolumnvalues
	SkewedColumnValues []string `json:"SkewedColumnValues,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSGluePartition_SkewedInfo) AWSCloudFormationType() string {
	return "AWS::Glue::Partition.SkewedInfo"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSGluePartition_SkewedInfo) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
