package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSSESConfigurationSetEventDestination AWS CloudFormation Resource (AWS::SES::ConfigurationSetEventDestination)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ses-configurationseteventdestination.html
type AWSSESConfigurationSetEventDestination struct {

	// ConfigurationSetName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ses-configurationseteventdestination.html#cfn-ses-configurationseteventdestination-configurationsetname
	ConfigurationSetName string `json:"ConfigurationSetName,omitempty"`

	// EventDestination AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ses-configurationseteventdestination.html#cfn-ses-configurationseteventdestination-eventdestination
	EventDestination *AWSSESConfigurationSetEventDestination_EventDestination `json:"EventDestination,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSESConfigurationSetEventDestination) AWSCloudFormationType() string {
	return "AWS::SES::ConfigurationSetEventDestination"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSESConfigurationSetEventDestination) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSSESConfigurationSetEventDestination) MarshalJSON() ([]byte, error) {
	type Properties AWSSESConfigurationSetEventDestination
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
func (r *AWSSESConfigurationSetEventDestination) UnmarshalJSON(b []byte) error {
	type Properties AWSSESConfigurationSetEventDestination
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
		*r = AWSSESConfigurationSetEventDestination(*res.Properties)
	}

	return nil
}

// GetAllAWSSESConfigurationSetEventDestinationResources retrieves all AWSSESConfigurationSetEventDestination items from an AWS CloudFormation template
func (t *Template) GetAllAWSSESConfigurationSetEventDestinationResources() map[string]AWSSESConfigurationSetEventDestination {
	results := map[string]AWSSESConfigurationSetEventDestination{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSSESConfigurationSetEventDestination:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::SES::ConfigurationSetEventDestination" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSSESConfigurationSetEventDestination
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

// GetAWSSESConfigurationSetEventDestinationWithName retrieves all AWSSESConfigurationSetEventDestination items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSSESConfigurationSetEventDestinationWithName(name string) (AWSSESConfigurationSetEventDestination, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSSESConfigurationSetEventDestination:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::SES::ConfigurationSetEventDestination" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSSESConfigurationSetEventDestination
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSSESConfigurationSetEventDestination{}, errors.New("resource not found")
}
