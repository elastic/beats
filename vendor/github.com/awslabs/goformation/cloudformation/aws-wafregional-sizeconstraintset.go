package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSWAFRegionalSizeConstraintSet AWS CloudFormation Resource (AWS::WAFRegional::SizeConstraintSet)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-wafregional-sizeconstraintset.html
type AWSWAFRegionalSizeConstraintSet struct {

	// Name AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-wafregional-sizeconstraintset.html#cfn-wafregional-sizeconstraintset-name
	Name string `json:"Name,omitempty"`

	// SizeConstraints AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-wafregional-sizeconstraintset.html#cfn-wafregional-sizeconstraintset-sizeconstraints
	SizeConstraints []AWSWAFRegionalSizeConstraintSet_SizeConstraint `json:"SizeConstraints,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSWAFRegionalSizeConstraintSet) AWSCloudFormationType() string {
	return "AWS::WAFRegional::SizeConstraintSet"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSWAFRegionalSizeConstraintSet) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSWAFRegionalSizeConstraintSet) MarshalJSON() ([]byte, error) {
	type Properties AWSWAFRegionalSizeConstraintSet
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
func (r *AWSWAFRegionalSizeConstraintSet) UnmarshalJSON(b []byte) error {
	type Properties AWSWAFRegionalSizeConstraintSet
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
		*r = AWSWAFRegionalSizeConstraintSet(*res.Properties)
	}

	return nil
}

// GetAllAWSWAFRegionalSizeConstraintSetResources retrieves all AWSWAFRegionalSizeConstraintSet items from an AWS CloudFormation template
func (t *Template) GetAllAWSWAFRegionalSizeConstraintSetResources() map[string]AWSWAFRegionalSizeConstraintSet {
	results := map[string]AWSWAFRegionalSizeConstraintSet{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSWAFRegionalSizeConstraintSet:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::WAFRegional::SizeConstraintSet" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSWAFRegionalSizeConstraintSet
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

// GetAWSWAFRegionalSizeConstraintSetWithName retrieves all AWSWAFRegionalSizeConstraintSet items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSWAFRegionalSizeConstraintSetWithName(name string) (AWSWAFRegionalSizeConstraintSet, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSWAFRegionalSizeConstraintSet:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::WAFRegional::SizeConstraintSet" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSWAFRegionalSizeConstraintSet
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSWAFRegionalSizeConstraintSet{}, errors.New("resource not found")
}
