package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSWAFRegionalIPSet AWS CloudFormation Resource (AWS::WAFRegional::IPSet)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-wafregional-ipset.html
type AWSWAFRegionalIPSet struct {

	// IPSetDescriptors AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-wafregional-ipset.html#cfn-wafregional-ipset-ipsetdescriptors
	IPSetDescriptors []AWSWAFRegionalIPSet_IPSetDescriptor `json:"IPSetDescriptors,omitempty"`

	// Name AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-wafregional-ipset.html#cfn-wafregional-ipset-name
	Name string `json:"Name,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSWAFRegionalIPSet) AWSCloudFormationType() string {
	return "AWS::WAFRegional::IPSet"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSWAFRegionalIPSet) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSWAFRegionalIPSet) MarshalJSON() ([]byte, error) {
	type Properties AWSWAFRegionalIPSet
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
func (r *AWSWAFRegionalIPSet) UnmarshalJSON(b []byte) error {
	type Properties AWSWAFRegionalIPSet
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
		*r = AWSWAFRegionalIPSet(*res.Properties)
	}

	return nil
}

// GetAllAWSWAFRegionalIPSetResources retrieves all AWSWAFRegionalIPSet items from an AWS CloudFormation template
func (t *Template) GetAllAWSWAFRegionalIPSetResources() map[string]AWSWAFRegionalIPSet {
	results := map[string]AWSWAFRegionalIPSet{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSWAFRegionalIPSet:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::WAFRegional::IPSet" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSWAFRegionalIPSet
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

// GetAWSWAFRegionalIPSetWithName retrieves all AWSWAFRegionalIPSet items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSWAFRegionalIPSetWithName(name string) (AWSWAFRegionalIPSet, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSWAFRegionalIPSet:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::WAFRegional::IPSet" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSWAFRegionalIPSet
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSWAFRegionalIPSet{}, errors.New("resource not found")
}
