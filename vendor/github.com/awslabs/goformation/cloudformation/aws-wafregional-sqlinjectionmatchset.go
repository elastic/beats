package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSWAFRegionalSqlInjectionMatchSet AWS CloudFormation Resource (AWS::WAFRegional::SqlInjectionMatchSet)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-wafregional-sqlinjectionmatchset.html
type AWSWAFRegionalSqlInjectionMatchSet struct {

	// Name AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-wafregional-sqlinjectionmatchset.html#cfn-wafregional-sqlinjectionmatchset-name
	Name string `json:"Name,omitempty"`

	// SqlInjectionMatchTuples AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-wafregional-sqlinjectionmatchset.html#cfn-wafregional-sqlinjectionmatchset-sqlinjectionmatchtuples
	SqlInjectionMatchTuples []AWSWAFRegionalSqlInjectionMatchSet_SqlInjectionMatchTuple `json:"SqlInjectionMatchTuples,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSWAFRegionalSqlInjectionMatchSet) AWSCloudFormationType() string {
	return "AWS::WAFRegional::SqlInjectionMatchSet"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSWAFRegionalSqlInjectionMatchSet) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSWAFRegionalSqlInjectionMatchSet) MarshalJSON() ([]byte, error) {
	type Properties AWSWAFRegionalSqlInjectionMatchSet
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
func (r *AWSWAFRegionalSqlInjectionMatchSet) UnmarshalJSON(b []byte) error {
	type Properties AWSWAFRegionalSqlInjectionMatchSet
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
		*r = AWSWAFRegionalSqlInjectionMatchSet(*res.Properties)
	}

	return nil
}

// GetAllAWSWAFRegionalSqlInjectionMatchSetResources retrieves all AWSWAFRegionalSqlInjectionMatchSet items from an AWS CloudFormation template
func (t *Template) GetAllAWSWAFRegionalSqlInjectionMatchSetResources() map[string]AWSWAFRegionalSqlInjectionMatchSet {
	results := map[string]AWSWAFRegionalSqlInjectionMatchSet{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSWAFRegionalSqlInjectionMatchSet:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::WAFRegional::SqlInjectionMatchSet" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSWAFRegionalSqlInjectionMatchSet
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

// GetAWSWAFRegionalSqlInjectionMatchSetWithName retrieves all AWSWAFRegionalSqlInjectionMatchSet items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSWAFRegionalSqlInjectionMatchSetWithName(name string) (AWSWAFRegionalSqlInjectionMatchSet, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSWAFRegionalSqlInjectionMatchSet:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::WAFRegional::SqlInjectionMatchSet" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSWAFRegionalSqlInjectionMatchSet
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSWAFRegionalSqlInjectionMatchSet{}, errors.New("resource not found")
}
