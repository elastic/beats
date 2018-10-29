package cloudformation

// AWSGlueTable_SkewedInfo AWS CloudFormation Resource (AWS::Glue::Table.SkewedInfo)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-table-skewedinfo.html
type AWSGlueTable_SkewedInfo struct {

	// SkewedColumnNames AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-table-skewedinfo.html#cfn-glue-table-skewedinfo-skewedcolumnnames
	SkewedColumnNames []string `json:"SkewedColumnNames,omitempty"`

	// SkewedColumnValueLocationMaps AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-table-skewedinfo.html#cfn-glue-table-skewedinfo-skewedcolumnvaluelocationmaps
	SkewedColumnValueLocationMaps interface{} `json:"SkewedColumnValueLocationMaps,omitempty"`

	// SkewedColumnValues AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-table-skewedinfo.html#cfn-glue-table-skewedinfo-skewedcolumnvalues
	SkewedColumnValues []string `json:"SkewedColumnValues,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSGlueTable_SkewedInfo) AWSCloudFormationType() string {
	return "AWS::Glue::Table.SkewedInfo"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSGlueTable_SkewedInfo) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
