package elasticsearch

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Domain_LogPublishingOption AWS CloudFormation Resource (AWS::Elasticsearch::Domain.LogPublishingOption)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticsearch-domain-logpublishingoption.html
type Domain_LogPublishingOption struct {

	// CloudWatchLogsLogGroupArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticsearch-domain-logpublishingoption.html#cfn-elasticsearch-domain-logpublishingoption-cloudwatchlogsloggrouparn
	CloudWatchLogsLogGroupArn string `json:"CloudWatchLogsLogGroupArn,omitempty"`

	// Enabled AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticsearch-domain-logpublishingoption.html#cfn-elasticsearch-domain-logpublishingoption-enabled
	Enabled bool `json:"Enabled,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Domain_LogPublishingOption) AWSCloudFormationType() string {
	return "AWS::Elasticsearch::Domain.LogPublishingOption"
}
