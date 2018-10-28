package cloudformation

// AWSKinesisAnalyticsApplicationReferenceDataSource_MappingParameters AWS CloudFormation Resource (AWS::KinesisAnalytics::ApplicationReferenceDataSource.MappingParameters)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationreferencedatasource-mappingparameters.html
type AWSKinesisAnalyticsApplicationReferenceDataSource_MappingParameters struct {

	// CSVMappingParameters AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationreferencedatasource-mappingparameters.html#cfn-kinesisanalytics-applicationreferencedatasource-mappingparameters-csvmappingparameters
	CSVMappingParameters *AWSKinesisAnalyticsApplicationReferenceDataSource_CSVMappingParameters `json:"CSVMappingParameters,omitempty"`

	// JSONMappingParameters AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationreferencedatasource-mappingparameters.html#cfn-kinesisanalytics-applicationreferencedatasource-mappingparameters-jsonmappingparameters
	JSONMappingParameters *AWSKinesisAnalyticsApplicationReferenceDataSource_JSONMappingParameters `json:"JSONMappingParameters,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSKinesisAnalyticsApplicationReferenceDataSource_MappingParameters) AWSCloudFormationType() string {
	return "AWS::KinesisAnalytics::ApplicationReferenceDataSource.MappingParameters"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSKinesisAnalyticsApplicationReferenceDataSource_MappingParameters) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
