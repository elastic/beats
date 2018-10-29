package cloudformation

// AWSKinesisAnalyticsApplication_InputSchema AWS CloudFormation Resource (AWS::KinesisAnalytics::Application.InputSchema)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-application-inputschema.html
type AWSKinesisAnalyticsApplication_InputSchema struct {

	// RecordColumns AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-application-inputschema.html#cfn-kinesisanalytics-application-inputschema-recordcolumns
	RecordColumns []AWSKinesisAnalyticsApplication_RecordColumn `json:"RecordColumns,omitempty"`

	// RecordEncoding AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-application-inputschema.html#cfn-kinesisanalytics-application-inputschema-recordencoding
	RecordEncoding string `json:"RecordEncoding,omitempty"`

	// RecordFormat AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-application-inputschema.html#cfn-kinesisanalytics-application-inputschema-recordformat
	RecordFormat *AWSKinesisAnalyticsApplication_RecordFormat `json:"RecordFormat,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSKinesisAnalyticsApplication_InputSchema) AWSCloudFormationType() string {
	return "AWS::KinesisAnalytics::Application.InputSchema"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSKinesisAnalyticsApplication_InputSchema) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
