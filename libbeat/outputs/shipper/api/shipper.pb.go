// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package api

import (
	reflect "reflect"
	sync "sync"

	status "google.golang.org/genproto/googleapis/rpc/status"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	structpb "google.golang.org/protobuf/types/known/structpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type PublishRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Events []*Event `protobuf:"bytes,1,rep,name=events,proto3" json:"events,omitempty"`
}

func (x *PublishRequest) Reset() {
	*x = PublishRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_shipper_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PublishRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PublishRequest) ProtoMessage() {}

func (x *PublishRequest) ProtoReflect() protoreflect.Message {
	mi := &file_shipper_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PublishRequest.ProtoReflect.Descriptor instead.
func (*PublishRequest) Descriptor() ([]byte, []int) {
	return file_shipper_proto_rawDescGZIP(), []int{0}
}

func (x *PublishRequest) GetEvents() []*Event {
	if x != nil {
		return x.Events
	}
	return nil
}

// Event is a translation of beat.Event into protobuf.
type Event struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Creation timestamp of the event.
	Timestamp *timestamppb.Timestamp `protobuf:"bytes,1,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	// Identifies the input that generated the event.
	Input *Input `protobuf:"bytes,2,opt,name=input,proto3" json:"input,omitempty"`
	// Optional. Producers may specify a unique event ID if the event has a natural unique
	// identifier to simplify acknowledgement tracking. If no such identifier exists, the queue
	// offset can be used for acknowledgement tracking.
	EventId string `protobuf:"bytes,4,opt,name=event_id,json=eventId,proto3" json:"event_id,omitempty"`
	// Data stream for the event.
	DataStream *DataStream `protobuf:"bytes,5,opt,name=data_stream,json=dataStream,proto3" json:"data_stream,omitempty"`
	// Metadata JSON object (map[string]google.protobuf.Value)
	Metadata *structpb.Struct `protobuf:"bytes,6,opt,name=metadata,proto3" json:"metadata,omitempty"`
	// Field JSON object (map[string]google.protobuf.Value)
	Fields *structpb.Struct `protobuf:"bytes,7,opt,name=fields,proto3" json:"fields,omitempty"`
}

func (x *Event) Reset() {
	*x = Event{}
	if protoimpl.UnsafeEnabled {
		mi := &file_shipper_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Event) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Event) ProtoMessage() {}

func (x *Event) ProtoReflect() protoreflect.Message {
	mi := &file_shipper_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Event.ProtoReflect.Descriptor instead.
func (*Event) Descriptor() ([]byte, []int) {
	return file_shipper_proto_rawDescGZIP(), []int{1}
}

func (x *Event) GetTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

func (x *Event) GetInput() *Input {
	if x != nil {
		return x.Input
	}
	return nil
}

func (x *Event) GetEventId() string {
	if x != nil {
		return x.EventId
	}
	return ""
}

func (x *Event) GetDataStream() *DataStream {
	if x != nil {
		return x.DataStream
	}
	return nil
}

func (x *Event) GetMetadata() *structpb.Struct {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *Event) GetFields() *structpb.Struct {
	if x != nil {
		return x.Fields
	}
	return nil
}

// Input identifiers from the agent policy, see example configuration.
type Input struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Input ID in the agent policy.
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// Input name in the agent policy.
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	// Input type in the agent policy (e.g. logfile, winlog, endpoint, etc.)
	Type string `protobuf:"bytes,3,opt,name=type,proto3" json:"type,omitempty"`
}

func (x *Input) Reset() {
	*x = Input{}
	if protoimpl.UnsafeEnabled {
		mi := &file_shipper_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Input) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Input) ProtoMessage() {}

func (x *Input) ProtoReflect() protoreflect.Message {
	mi := &file_shipper_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Input.ProtoReflect.Descriptor instead.
func (*Input) Descriptor() ([]byte, []int) {
	return file_shipper_proto_rawDescGZIP(), []int{2}
}

func (x *Input) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Input) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Input) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

// See https://www.elastic.co/blog/an-introduction-to-the-elastic-data-stream-naming-scheme
type DataStream struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Optional. Stable identifier for the data stream, see example configuration.
	Id        string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Type      string `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	Dataset   string `protobuf:"bytes,3,opt,name=dataset,proto3" json:"dataset,omitempty"`
	Namespace string `protobuf:"bytes,4,opt,name=namespace,proto3" json:"namespace,omitempty"`
}

func (x *DataStream) Reset() {
	*x = DataStream{}
	if protoimpl.UnsafeEnabled {
		mi := &file_shipper_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DataStream) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DataStream) ProtoMessage() {}

func (x *DataStream) ProtoReflect() protoreflect.Message {
	mi := &file_shipper_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DataStream.ProtoReflect.Descriptor instead.
func (*DataStream) Descriptor() ([]byte, []int) {
	return file_shipper_proto_rawDescGZIP(), []int{3}
}

func (x *DataStream) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *DataStream) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *DataStream) GetDataset() string {
	if x != nil {
		return x.Dataset
	}
	return ""
}

func (x *DataStream) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

type PublishReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Results describing messages successfully published to the queue.
	// The output order matches the input order e.g. results[N] maps to events[N].
	Results []*EventResult `protobuf:"bytes,1,rep,name=results,proto3" json:"results,omitempty"`
}

func (x *PublishReply) Reset() {
	*x = PublishReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_shipper_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PublishReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PublishReply) ProtoMessage() {}

func (x *PublishReply) ProtoReflect() protoreflect.Message {
	mi := &file_shipper_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PublishReply.ProtoReflect.Descriptor instead.
func (*PublishReply) Descriptor() ([]byte, []int) {
	return file_shipper_proto_rawDescGZIP(), []int{4}
}

func (x *PublishReply) GetResults() []*EventResult {
	if x != nil {
		return x.Results
	}
	return nil
}

type EventResult struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Timestamp of the event that was published.
	Timestamp *timestamppb.Timestamp `protobuf:"bytes,1,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	// ID derived from the queue offset that uniquely identifies the event within the shipper
	// system. Used to track the progress of events through the queue system.
	QueueId string `protobuf:"bytes,2,opt,name=queue_id,json=queueId,proto3" json:"queue_id,omitempty"`
	// Optional. Event ID provided when the event was published, if it exists.
	EventId string `protobuf:"bytes,3,opt,name=event_id,json=eventId,proto3" json:"event_id,omitempty"`
	// Optional. List of errors encountered by processors, if any. Processor
	// errors do not indicate the message failed to send.
	ProcessorErrors []*status.Status `protobuf:"bytes,5,rep,name=processor_errors,json=processorErrors,proto3" json:"processor_errors,omitempty"`
}

func (x *EventResult) Reset() {
	*x = EventResult{}
	if protoimpl.UnsafeEnabled {
		mi := &file_shipper_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventResult) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventResult) ProtoMessage() {}

func (x *EventResult) ProtoReflect() protoreflect.Message {
	mi := &file_shipper_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventResult.ProtoReflect.Descriptor instead.
func (*EventResult) Descriptor() ([]byte, []int) {
	return file_shipper_proto_rawDescGZIP(), []int{5}
}

func (x *EventResult) GetTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

func (x *EventResult) GetQueueId() string {
	if x != nil {
		return x.QueueId
	}
	return ""
}

func (x *EventResult) GetEventId() string {
	if x != nil {
		return x.EventId
	}
	return ""
}

func (x *EventResult) GetProcessorErrors() []*status.Status {
	if x != nil {
		return x.ProcessorErrors
	}
	return nil
}

type StreamAcksRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Optional. Requests acknowledgements originating only from this input ID.
	InputId string `protobuf:"bytes,1,opt,name=input_id,json=inputId,proto3" json:"input_id,omitempty"`
	// Optional. Requests acknowledgements for events for this data stream only.
	DataStream *DataStream `protobuf:"bytes,2,opt,name=data_stream,json=dataStream,proto3" json:"data_stream,omitempty"`
}

func (x *StreamAcksRequest) Reset() {
	*x = StreamAcksRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_shipper_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StreamAcksRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StreamAcksRequest) ProtoMessage() {}

func (x *StreamAcksRequest) ProtoReflect() protoreflect.Message {
	mi := &file_shipper_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StreamAcksRequest.ProtoReflect.Descriptor instead.
func (*StreamAcksRequest) Descriptor() ([]byte, []int) {
	return file_shipper_proto_rawDescGZIP(), []int{6}
}

func (x *StreamAcksRequest) GetInputId() string {
	if x != nil {
		return x.InputId
	}
	return ""
}

func (x *StreamAcksRequest) GetDataStream() *DataStream {
	if x != nil {
		return x.DataStream
	}
	return nil
}

type StreamAcksReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Acks []*Acknowledgement `protobuf:"bytes,1,rep,name=acks,proto3" json:"acks,omitempty"`
}

func (x *StreamAcksReply) Reset() {
	*x = StreamAcksReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_shipper_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StreamAcksReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StreamAcksReply) ProtoMessage() {}

func (x *StreamAcksReply) ProtoReflect() protoreflect.Message {
	mi := &file_shipper_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StreamAcksReply.ProtoReflect.Descriptor instead.
func (*StreamAcksReply) Descriptor() ([]byte, []int) {
	return file_shipper_proto_rawDescGZIP(), []int{7}
}

func (x *StreamAcksReply) GetAcks() []*Acknowledgement {
	if x != nil {
		return x.Acks
	}
	return nil
}

type Acknowledgement struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Timestamp of the event being acknowledged.
	Timestamp *timestamppb.Timestamp `protobuf:"bytes,1,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	// ID derived from the queue offset that uniquely identifies the event within the shipper
	// system. Used to track the progress of events through the queue system.
	QueueId string `protobuf:"bytes,2,opt,name=queue_id,json=queueId,proto3" json:"queue_id,omitempty"`
	// Optional. Event ID provided when the event was published, if it exists.
	EventId string `protobuf:"bytes,3,opt,name=event_id,json=eventId,proto3" json:"event_id,omitempty"`
	// Optional. Error status indicating the message failed to send.
	Error *status.Status `protobuf:"bytes,4,opt,name=error,proto3" json:"error,omitempty"`
}

func (x *Acknowledgement) Reset() {
	*x = Acknowledgement{}
	if protoimpl.UnsafeEnabled {
		mi := &file_shipper_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Acknowledgement) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Acknowledgement) ProtoMessage() {}

func (x *Acknowledgement) ProtoReflect() protoreflect.Message {
	mi := &file_shipper_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Acknowledgement.ProtoReflect.Descriptor instead.
func (*Acknowledgement) Descriptor() ([]byte, []int) {
	return file_shipper_proto_rawDescGZIP(), []int{8}
}

func (x *Acknowledgement) GetTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

func (x *Acknowledgement) GetQueueId() string {
	if x != nil {
		return x.QueueId
	}
	return ""
}

func (x *Acknowledgement) GetEventId() string {
	if x != nil {
		return x.EventId
	}
	return ""
}

func (x *Acknowledgement) GetError() *status.Status {
	if x != nil {
		return x.Error
	}
	return nil
}

var File_shipper_proto protoreflect.FileDescriptor

var file_shipper_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x73, 0x68, 0x69, 0x70, 0x70, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x18, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x73,
	0x68, 0x69, 0x70, 0x70, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x73, 0x74, 0x72, 0x75,
	0x63, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x17, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2f, 0x72, 0x70, 0x63, 0x2f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0x49, 0x0a, 0x0e, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x73, 0x68, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x37, 0x0a, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x18, 0x01, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x67,
	0x65, 0x6e, 0x74, 0x2e, 0x73, 0x68, 0x69, 0x70, 0x70, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x45,
	0x76, 0x65, 0x6e, 0x74, 0x52, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x22, 0xc0, 0x02, 0x0a,
	0x05, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x12, 0x38, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65,
	0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x12, 0x35, 0x0a, 0x05, 0x69, 0x6e, 0x70, 0x75, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1f, 0x2e, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e,
	0x73, 0x68, 0x69, 0x70, 0x70, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x49, 0x6e, 0x70, 0x75, 0x74,
	0x52, 0x05, 0x69, 0x6e, 0x70, 0x75, 0x74, 0x12, 0x19, 0x0a, 0x08, 0x65, 0x76, 0x65, 0x6e, 0x74,
	0x5f, 0x69, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x65, 0x76, 0x65, 0x6e, 0x74,
	0x49, 0x64, 0x12, 0x45, 0x0a, 0x0b, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x73, 0x74, 0x72, 0x65, 0x61,
	0x6d, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69,
	0x63, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x73, 0x68, 0x69, 0x70, 0x70, 0x65, 0x72, 0x2e,
	0x76, 0x31, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x52, 0x0a, 0x64,
	0x61, 0x74, 0x61, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x12, 0x33, 0x0a, 0x08, 0x6d, 0x65, 0x74,
	0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74,
	0x72, 0x75, 0x63, 0x74, 0x52, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x2f,
	0x0a, 0x06, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x73, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x52, 0x06, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x73, 0x22,
	0x3f, 0x0a, 0x05, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x12, 0x0a, 0x04,
	0x74, 0x79, 0x70, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65,
	0x22, 0x68, 0x0a, 0x0a, 0x44, 0x61, 0x74, 0x61, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x12, 0x0e,
	0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12,
	0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79,
	0x70, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x64, 0x61, 0x74, 0x61, 0x73, 0x65, 0x74, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x07, 0x64, 0x61, 0x74, 0x61, 0x73, 0x65, 0x74, 0x12, 0x1c, 0x0a, 0x09,
	0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x22, 0x4f, 0x0a, 0x0c, 0x50, 0x75,
	0x62, 0x6c, 0x69, 0x73, 0x68, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x3f, 0x0a, 0x07, 0x72, 0x65,
	0x73, 0x75, 0x6c, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x25, 0x2e, 0x65, 0x6c,
	0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x73, 0x68, 0x69, 0x70,
	0x70, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x52, 0x65, 0x73, 0x75,
	0x6c, 0x74, 0x52, 0x07, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x73, 0x22, 0xbc, 0x01, 0x0a, 0x0b,
	0x45, 0x76, 0x65, 0x6e, 0x74, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x12, 0x38, 0x0a, 0x09, 0x74,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65,
	0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x19, 0x0a, 0x08, 0x71, 0x75, 0x65, 0x75, 0x65, 0x5f, 0x69,
	0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x71, 0x75, 0x65, 0x75, 0x65, 0x49, 0x64,
	0x12, 0x19, 0x0a, 0x08, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x07, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x12, 0x3d, 0x0a, 0x10, 0x70,
	0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x6f, 0x72, 0x5f, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x73, 0x18,
	0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x72,
	0x70, 0x63, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x0f, 0x70, 0x72, 0x6f, 0x63, 0x65,
	0x73, 0x73, 0x6f, 0x72, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x73, 0x22, 0x75, 0x0a, 0x11, 0x53, 0x74,
	0x72, 0x65, 0x61, 0x6d, 0x41, 0x63, 0x6b, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x19, 0x0a, 0x08, 0x69, 0x6e, 0x70, 0x75, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x07, 0x69, 0x6e, 0x70, 0x75, 0x74, 0x49, 0x64, 0x12, 0x45, 0x0a, 0x0b, 0x64, 0x61,
	0x74, 0x61, 0x5f, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x24, 0x2e, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e,
	0x73, 0x68, 0x69, 0x70, 0x70, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x53,
	0x74, 0x72, 0x65, 0x61, 0x6d, 0x52, 0x0a, 0x64, 0x61, 0x74, 0x61, 0x53, 0x74, 0x72, 0x65, 0x61,
	0x6d, 0x22, 0x50, 0x0a, 0x0f, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x41, 0x63, 0x6b, 0x73, 0x52,
	0x65, 0x70, 0x6c, 0x79, 0x12, 0x3d, 0x0a, 0x04, 0x61, 0x63, 0x6b, 0x73, 0x18, 0x01, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x29, 0x2e, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x67, 0x65,
	0x6e, 0x74, 0x2e, 0x73, 0x68, 0x69, 0x70, 0x70, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x41, 0x63,
	0x6b, 0x6e, 0x6f, 0x77, 0x6c, 0x65, 0x64, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x52, 0x04, 0x61,
	0x63, 0x6b, 0x73, 0x22, 0xab, 0x01, 0x0a, 0x0f, 0x41, 0x63, 0x6b, 0x6e, 0x6f, 0x77, 0x6c, 0x65,
	0x64, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x12, 0x38, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d,
	0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x12, 0x19, 0x0a, 0x08, 0x71, 0x75, 0x65, 0x75, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x07, 0x71, 0x75, 0x65, 0x75, 0x65, 0x49, 0x64, 0x12, 0x19, 0x0a, 0x08,
	0x65, 0x76, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07,
	0x65, 0x76, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x12, 0x28, 0x0a, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x72, 0x70, 0x63, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x05, 0x65, 0x72, 0x72, 0x6f,
	0x72, 0x32, 0xe1, 0x01, 0x0a, 0x08, 0x50, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x65, 0x72, 0x12, 0x61,
	0x0a, 0x0d, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x73, 0x68, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x12,
	0x28, 0x2e, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e,
	0x73, 0x68, 0x69, 0x70, 0x70, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x75, 0x62, 0x6c, 0x69,
	0x73, 0x68, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x26, 0x2e, 0x65, 0x6c, 0x61, 0x73,
	0x74, 0x69, 0x63, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x73, 0x68, 0x69, 0x70, 0x70, 0x65,
	0x72, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x73, 0x68, 0x52, 0x65, 0x70, 0x6c,
	0x79, 0x12, 0x72, 0x0a, 0x16, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x41, 0x63, 0x6b, 0x6e, 0x6f,
	0x77, 0x6c, 0x65, 0x64, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x73, 0x12, 0x2b, 0x2e, 0x65, 0x6c,
	0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x73, 0x68, 0x69, 0x70,
	0x70, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x41, 0x63, 0x6b,
	0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x29, 0x2e, 0x65, 0x6c, 0x61, 0x73, 0x74,
	0x69, 0x63, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x73, 0x68, 0x69, 0x70, 0x70, 0x65, 0x72,
	0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x41, 0x63, 0x6b, 0x73, 0x52, 0x65,
	0x70, 0x6c, 0x79, 0x30, 0x01, 0x42, 0x2e, 0x5a, 0x2c, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2f, 0x65, 0x6c, 0x61, 0x73,
	0x74, 0x69, 0x63, 0x2d, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2d, 0x73, 0x68, 0x69, 0x70, 0x70, 0x65,
	0x72, 0x2f, 0x61, 0x70, 0x69, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_shipper_proto_rawDescOnce sync.Once
	file_shipper_proto_rawDescData = file_shipper_proto_rawDesc
)

func file_shipper_proto_rawDescGZIP() []byte {
	file_shipper_proto_rawDescOnce.Do(func() {
		file_shipper_proto_rawDescData = protoimpl.X.CompressGZIP(file_shipper_proto_rawDescData)
	})
	return file_shipper_proto_rawDescData
}

var file_shipper_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_shipper_proto_goTypes = []interface{}{
	(*PublishRequest)(nil),        // 0: elastic.agent.shipper.v1.PublishRequest
	(*Event)(nil),                 // 1: elastic.agent.shipper.v1.Event
	(*Input)(nil),                 // 2: elastic.agent.shipper.v1.Input
	(*DataStream)(nil),            // 3: elastic.agent.shipper.v1.DataStream
	(*PublishReply)(nil),          // 4: elastic.agent.shipper.v1.PublishReply
	(*EventResult)(nil),           // 5: elastic.agent.shipper.v1.EventResult
	(*StreamAcksRequest)(nil),     // 6: elastic.agent.shipper.v1.StreamAcksRequest
	(*StreamAcksReply)(nil),       // 7: elastic.agent.shipper.v1.StreamAcksReply
	(*Acknowledgement)(nil),       // 8: elastic.agent.shipper.v1.Acknowledgement
	(*timestamppb.Timestamp)(nil), // 9: google.protobuf.Timestamp
	(*structpb.Struct)(nil),       // 10: google.protobuf.Struct
	(*status.Status)(nil),         // 11: google.rpc.Status
}
var file_shipper_proto_depIdxs = []int32{
	1,  // 0: elastic.agent.shipper.v1.PublishRequest.events:type_name -> elastic.agent.shipper.v1.Event
	9,  // 1: elastic.agent.shipper.v1.Event.timestamp:type_name -> google.protobuf.Timestamp
	2,  // 2: elastic.agent.shipper.v1.Event.input:type_name -> elastic.agent.shipper.v1.Input
	3,  // 3: elastic.agent.shipper.v1.Event.data_stream:type_name -> elastic.agent.shipper.v1.DataStream
	10, // 4: elastic.agent.shipper.v1.Event.metadata:type_name -> google.protobuf.Struct
	10, // 5: elastic.agent.shipper.v1.Event.fields:type_name -> google.protobuf.Struct
	5,  // 6: elastic.agent.shipper.v1.PublishReply.results:type_name -> elastic.agent.shipper.v1.EventResult
	9,  // 7: elastic.agent.shipper.v1.EventResult.timestamp:type_name -> google.protobuf.Timestamp
	11, // 8: elastic.agent.shipper.v1.EventResult.processor_errors:type_name -> google.rpc.Status
	3,  // 9: elastic.agent.shipper.v1.StreamAcksRequest.data_stream:type_name -> elastic.agent.shipper.v1.DataStream
	8,  // 10: elastic.agent.shipper.v1.StreamAcksReply.acks:type_name -> elastic.agent.shipper.v1.Acknowledgement
	9,  // 11: elastic.agent.shipper.v1.Acknowledgement.timestamp:type_name -> google.protobuf.Timestamp
	11, // 12: elastic.agent.shipper.v1.Acknowledgement.error:type_name -> google.rpc.Status
	0,  // 13: elastic.agent.shipper.v1.Producer.PublishEvents:input_type -> elastic.agent.shipper.v1.PublishRequest
	6,  // 14: elastic.agent.shipper.v1.Producer.StreamAcknowledgements:input_type -> elastic.agent.shipper.v1.StreamAcksRequest
	4,  // 15: elastic.agent.shipper.v1.Producer.PublishEvents:output_type -> elastic.agent.shipper.v1.PublishReply
	7,  // 16: elastic.agent.shipper.v1.Producer.StreamAcknowledgements:output_type -> elastic.agent.shipper.v1.StreamAcksReply
	15, // [15:17] is the sub-list for method output_type
	13, // [13:15] is the sub-list for method input_type
	13, // [13:13] is the sub-list for extension type_name
	13, // [13:13] is the sub-list for extension extendee
	0,  // [0:13] is the sub-list for field type_name
}

func init() { file_shipper_proto_init() }
func file_shipper_proto_init() {
	if File_shipper_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_shipper_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PublishRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_shipper_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Event); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_shipper_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Input); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_shipper_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DataStream); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_shipper_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PublishReply); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_shipper_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventResult); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_shipper_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StreamAcksRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_shipper_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StreamAcksReply); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_shipper_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Acknowledgement); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_shipper_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_shipper_proto_goTypes,
		DependencyIndexes: file_shipper_proto_depIdxs,
		MessageInfos:      file_shipper_proto_msgTypes,
	}.Build()
	File_shipper_proto = out.File
	file_shipper_proto_rawDesc = nil
	file_shipper_proto_goTypes = nil
	file_shipper_proto_depIdxs = nil
}
