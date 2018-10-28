package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSServiceCatalogTagOptionAssociation AWS CloudFormation Resource (AWS::ServiceCatalog::TagOptionAssociation)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-servicecatalog-tagoptionassociation.html
type AWSServiceCatalogTagOptionAssociation struct {

	// ResourceId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-servicecatalog-tagoptionassociation.html#cfn-servicecatalog-tagoptionassociation-resourceid
	ResourceId string `json:"ResourceId,omitempty"`

	// TagOptionId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-servicecatalog-tagoptionassociation.html#cfn-servicecatalog-tagoptionassociation-tagoptionid
	TagOptionId string `json:"TagOptionId,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSServiceCatalogTagOptionAssociation) AWSCloudFormationType() string {
	return "AWS::ServiceCatalog::TagOptionAssociation"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSServiceCatalogTagOptionAssociation) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSServiceCatalogTagOptionAssociation) MarshalJSON() ([]byte, error) {
	type Properties AWSServiceCatalogTagOptionAssociation
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
func (r *AWSServiceCatalogTagOptionAssociation) UnmarshalJSON(b []byte) error {
	type Properties AWSServiceCatalogTagOptionAssociation
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
		*r = AWSServiceCatalogTagOptionAssociation(*res.Properties)
	}

	return nil
}

// GetAllAWSServiceCatalogTagOptionAssociationResources retrieves all AWSServiceCatalogTagOptionAssociation items from an AWS CloudFormation template
func (t *Template) GetAllAWSServiceCatalogTagOptionAssociationResources() map[string]AWSServiceCatalogTagOptionAssociation {
	results := map[string]AWSServiceCatalogTagOptionAssociation{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSServiceCatalogTagOptionAssociation:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::ServiceCatalog::TagOptionAssociation" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSServiceCatalogTagOptionAssociation
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

// GetAWSServiceCatalogTagOptionAssociationWithName retrieves all AWSServiceCatalogTagOptionAssociation items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSServiceCatalogTagOptionAssociationWithName(name string) (AWSServiceCatalogTagOptionAssociation, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSServiceCatalogTagOptionAssociation:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::ServiceCatalog::TagOptionAssociation" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSServiceCatalogTagOptionAssociation
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSServiceCatalogTagOptionAssociation{}, errors.New("resource not found")
}
