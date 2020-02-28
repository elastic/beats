package glue

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Crawler_SchemaChangePolicy AWS CloudFormation Resource (AWS::Glue::Crawler.SchemaChangePolicy)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-crawler-schemachangepolicy.html
type Crawler_SchemaChangePolicy struct {

	// DeleteBehavior AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-crawler-schemachangepolicy.html#cfn-glue-crawler-schemachangepolicy-deletebehavior
	DeleteBehavior string `json:"DeleteBehavior,omitempty"`

	// UpdateBehavior AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-crawler-schemachangepolicy.html#cfn-glue-crawler-schemachangepolicy-updatebehavior
	UpdateBehavior string `json:"UpdateBehavior,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Crawler_SchemaChangePolicy) AWSCloudFormationType() string {
	return "AWS::Glue::Crawler.SchemaChangePolicy"
}
