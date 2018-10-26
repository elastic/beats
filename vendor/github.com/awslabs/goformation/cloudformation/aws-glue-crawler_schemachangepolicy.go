package cloudformation

// AWSGlueCrawler_SchemaChangePolicy AWS CloudFormation Resource (AWS::Glue::Crawler.SchemaChangePolicy)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-crawler-schemachangepolicy.html
type AWSGlueCrawler_SchemaChangePolicy struct {

	// DeleteBehavior AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-crawler-schemachangepolicy.html#cfn-glue-crawler-schemachangepolicy-deletebehavior
	DeleteBehavior string `json:"DeleteBehavior,omitempty"`

	// UpdateBehavior AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-crawler-schemachangepolicy.html#cfn-glue-crawler-schemachangepolicy-updatebehavior
	UpdateBehavior string `json:"UpdateBehavior,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSGlueCrawler_SchemaChangePolicy) AWSCloudFormationType() string {
	return "AWS::Glue::Crawler.SchemaChangePolicy"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSGlueCrawler_SchemaChangePolicy) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
