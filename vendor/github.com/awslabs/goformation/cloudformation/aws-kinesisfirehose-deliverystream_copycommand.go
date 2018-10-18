package cloudformation

// AWSKinesisFirehoseDeliveryStream_CopyCommand AWS CloudFormation Resource (AWS::KinesisFirehose::DeliveryStream.CopyCommand)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-copycommand.html
type AWSKinesisFirehoseDeliveryStream_CopyCommand struct {

	// CopyOptions AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-copycommand.html#cfn-kinesisfirehose-deliverystream-copycommand-copyoptions
	CopyOptions string `json:"CopyOptions,omitempty"`

	// DataTableColumns AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-copycommand.html#cfn-kinesisfirehose-deliverystream-copycommand-datatablecolumns
	DataTableColumns string `json:"DataTableColumns,omitempty"`

	// DataTableName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-copycommand.html#cfn-kinesisfirehose-deliverystream-copycommand-datatablename
	DataTableName string `json:"DataTableName,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSKinesisFirehoseDeliveryStream_CopyCommand) AWSCloudFormationType() string {
	return "AWS::KinesisFirehose::DeliveryStream.CopyCommand"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSKinesisFirehoseDeliveryStream_CopyCommand) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
