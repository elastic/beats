package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSGlueCrawler AWS CloudFormation Resource (AWS::Glue::Crawler)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-glue-crawler.html
type AWSGlueCrawler struct {

	// Classifiers AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-glue-crawler.html#cfn-glue-crawler-classifiers
	Classifiers []string `json:"Classifiers,omitempty"`

	// Configuration AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-glue-crawler.html#cfn-glue-crawler-configuration
	Configuration string `json:"Configuration,omitempty"`

	// DatabaseName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-glue-crawler.html#cfn-glue-crawler-databasename
	DatabaseName string `json:"DatabaseName,omitempty"`

	// Description AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-glue-crawler.html#cfn-glue-crawler-description
	Description string `json:"Description,omitempty"`

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-glue-crawler.html#cfn-glue-crawler-name
	Name string `json:"Name,omitempty"`

	// Role AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-glue-crawler.html#cfn-glue-crawler-role
	Role string `json:"Role,omitempty"`

	// Schedule AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-glue-crawler.html#cfn-glue-crawler-schedule
	Schedule *AWSGlueCrawler_Schedule `json:"Schedule,omitempty"`

	// SchemaChangePolicy AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-glue-crawler.html#cfn-glue-crawler-schemachangepolicy
	SchemaChangePolicy *AWSGlueCrawler_SchemaChangePolicy `json:"SchemaChangePolicy,omitempty"`

	// TablePrefix AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-glue-crawler.html#cfn-glue-crawler-tableprefix
	TablePrefix string `json:"TablePrefix,omitempty"`

	// Targets AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-glue-crawler.html#cfn-glue-crawler-targets
	Targets *AWSGlueCrawler_Targets `json:"Targets,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSGlueCrawler) AWSCloudFormationType() string {
	return "AWS::Glue::Crawler"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSGlueCrawler) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSGlueCrawler) MarshalJSON() ([]byte, error) {
	type Properties AWSGlueCrawler
	return json.Marshal(&struct {
		Type           string
		Properties     Properties
		DeletionPolicy DeletionPolicy `json:"DeletionPolicy,omitempty"`
	}{
		Type:           r.AWSCloudFormationType(),
		Properties:     (Properties)(r),
		DeletionPolicy: r._deletionPolicy,
	})
}

// UnmarshalJSON is a custom JSON unmarshalling hook that strips the outer
// AWS CloudFormation resource object, and just keeps the 'Properties' field.
func (r *AWSGlueCrawler) UnmarshalJSON(b []byte) error {
	type Properties AWSGlueCrawler
	res := &struct {
		Type       string
		Properties *Properties
	}{}
	if err := json.Unmarshal(b, &res); err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return err
	}

	// If the resource has no Properties set, it could be nil
	if res.Properties != nil {
		*r = AWSGlueCrawler(*res.Properties)
	}

	return nil
}

// GetAllAWSGlueCrawlerResources retrieves all AWSGlueCrawler items from an AWS CloudFormation template
func (t *Template) GetAllAWSGlueCrawlerResources() map[string]AWSGlueCrawler {
	results := map[string]AWSGlueCrawler{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSGlueCrawler:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Glue::Crawler" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSGlueCrawler
						if err := json.Unmarshal(b, &result); err == nil {
							results[name] = result
						}
					}
				}
			}
		}
	}
	return results
}

// GetAWSGlueCrawlerWithName retrieves all AWSGlueCrawler items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSGlueCrawlerWithName(name string) (AWSGlueCrawler, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSGlueCrawler:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Glue::Crawler" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSGlueCrawler
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSGlueCrawler{}, errors.New("resource not found")
}
