package cloudformation

// AWSKinesisAnalyticsApplicationReferenceDataSource_RecordFormat AWS CloudFormation Resource (AWS::KinesisAnalytics::ApplicationReferenceDataSource.RecordFormat)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationreferencedatasource-recordformat.html
type AWSKinesisAnalyticsApplicationReferenceDataSource_RecordFormat struct {

	// MappingParameters AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationreferencedatasource-recordformat.html#cfn-kinesisanalytics-applicationreferencedatasource-recordformat-mappingparameters
	MappingParameters *AWSKinesisAnalyticsApplicationReferenceDataSource_MappingParameters `json:"MappingParameters,omitempty"`

	// RecordFormatType AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationreferencedatasource-recordformat.html#cfn-kinesisanalytics-applicationreferencedatasource-recordformat-recordformattype
	RecordFormatType string `json:"RecordFormatType,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSKinesisAnalyticsApplicationReferenceDataSource_RecordFormat) AWSCloudFormationType() string {
	return "AWS::KinesisAnalytics::ApplicationReferenceDataSource.RecordFormat"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSKinesisAnalyticsApplicationReferenceDataSource_RecordFormat) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
