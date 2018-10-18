package cloudformation

import (
	"encoding/json"

	"reflect"

	"github.com/mitchellh/mapstructure"
)

// AWSServerlessFunction_Properties is a helper struct that can hold either a S3Event, SNSEvent, SQSEvent, KinesisEvent, DynamoDBEvent, ApiEvent, ScheduleEvent, CloudWatchEventEvent, IoTRuleEvent, or AlexaSkillEvent value
type AWSServerlessFunction_Properties struct {
	S3Event              *AWSServerlessFunction_S3Event
	SNSEvent             *AWSServerlessFunction_SNSEvent
	SQSEvent             *AWSServerlessFunction_SQSEvent
	KinesisEvent         *AWSServerlessFunction_KinesisEvent
	DynamoDBEvent        *AWSServerlessFunction_DynamoDBEvent
	ApiEvent             *AWSServerlessFunction_ApiEvent
	ScheduleEvent        *AWSServerlessFunction_ScheduleEvent
	CloudWatchEventEvent *AWSServerlessFunction_CloudWatchEventEvent
	IoTRuleEvent         *AWSServerlessFunction_IoTRuleEvent
	AlexaSkillEvent      *AWSServerlessFunction_AlexaSkillEvent
}

func (r AWSServerlessFunction_Properties) value() interface{} {

	if r.S3Event != nil && !reflect.DeepEqual(r.S3Event, &AWSServerlessFunction_S3Event{}) {
		return r.S3Event
	}

	if r.SNSEvent != nil && !reflect.DeepEqual(r.SNSEvent, &AWSServerlessFunction_SNSEvent{}) {
		return r.SNSEvent
	}

	if r.SQSEvent != nil && !reflect.DeepEqual(r.SQSEvent, &AWSServerlessFunction_SQSEvent{}) {
		return r.SQSEvent
	}

	if r.KinesisEvent != nil && !reflect.DeepEqual(r.KinesisEvent, &AWSServerlessFunction_KinesisEvent{}) {
		return r.KinesisEvent
	}

	if r.DynamoDBEvent != nil && !reflect.DeepEqual(r.DynamoDBEvent, &AWSServerlessFunction_DynamoDBEvent{}) {
		return r.DynamoDBEvent
	}

	if r.ApiEvent != nil && !reflect.DeepEqual(r.ApiEvent, &AWSServerlessFunction_ApiEvent{}) {
		return r.ApiEvent
	}

	if r.ScheduleEvent != nil && !reflect.DeepEqual(r.ScheduleEvent, &AWSServerlessFunction_ScheduleEvent{}) {
		return r.ScheduleEvent
	}

	if r.CloudWatchEventEvent != nil && !reflect.DeepEqual(r.CloudWatchEventEvent, &AWSServerlessFunction_CloudWatchEventEvent{}) {
		return r.CloudWatchEventEvent
	}

	if r.IoTRuleEvent != nil && !reflect.DeepEqual(r.IoTRuleEvent, &AWSServerlessFunction_IoTRuleEvent{}) {
		return r.IoTRuleEvent
	}

	if r.AlexaSkillEvent != nil && !reflect.DeepEqual(r.AlexaSkillEvent, &AWSServerlessFunction_AlexaSkillEvent{}) {
		return r.AlexaSkillEvent
	}

	if r.S3Event != nil {
		return r.S3Event
	}

	if r.SNSEvent != nil {
		return r.SNSEvent
	}

	if r.SQSEvent != nil {
		return r.SQSEvent
	}

	if r.KinesisEvent != nil {
		return r.KinesisEvent
	}

	if r.DynamoDBEvent != nil {
		return r.DynamoDBEvent
	}

	if r.ApiEvent != nil {
		return r.ApiEvent
	}

	if r.ScheduleEvent != nil {
		return r.ScheduleEvent
	}

	if r.CloudWatchEventEvent != nil {
		return r.CloudWatchEventEvent
	}

	if r.IoTRuleEvent != nil {
		return r.IoTRuleEvent
	}

	if r.AlexaSkillEvent != nil {
		return r.AlexaSkillEvent
	}

	return nil

}

func (r AWSServerlessFunction_Properties) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.value())
}

// Hook into the marshaller
func (r *AWSServerlessFunction_Properties) UnmarshalJSON(b []byte) error {

	// Unmarshal into interface{} to check it's type
	var typecheck interface{}
	if err := json.Unmarshal(b, &typecheck); err != nil {
		return err
	}

	switch val := typecheck.(type) {

	case map[string]interface{}:

		mapstructure.Decode(val, &r.S3Event)

		mapstructure.Decode(val, &r.SNSEvent)

		mapstructure.Decode(val, &r.SQSEvent)

		mapstructure.Decode(val, &r.KinesisEvent)

		mapstructure.Decode(val, &r.DynamoDBEvent)

		mapstructure.Decode(val, &r.ApiEvent)

		mapstructure.Decode(val, &r.ScheduleEvent)

		mapstructure.Decode(val, &r.CloudWatchEventEvent)

		mapstructure.Decode(val, &r.IoTRuleEvent)

		mapstructure.Decode(val, &r.AlexaSkillEvent)

	case []interface{}:

	}

	return nil
}
