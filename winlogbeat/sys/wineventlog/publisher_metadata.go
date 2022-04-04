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

//go:build windows
// +build windows

package wineventlog

import (
	"os"
	"syscall"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/sys/windows"
)

// PublisherMetadata provides methods to query metadata from an event log
// publisher.
type PublisherMetadata struct {
	Name   string    // Name of the publisher/provider.
	Handle EvtHandle // Handle to the publisher metadata from EvtOpenPublisherMetadata.
}

// Close releases the publisher metadata handle.
func (m *PublisherMetadata) Close() error {
	return m.Handle.Close()
}

// NewPublisherMetadata opens the publisher's metadata. Close must be called on
// the returned PublisherMetadata to release its handle.
func NewPublisherMetadata(session EvtHandle, name string) (*PublisherMetadata, error) {
	var publisherName, logFile *uint16
	if info, err := os.Stat(name); err == nil && info.Mode().IsRegular() {
		logFile, err = syscall.UTF16PtrFromString(name)
		if err != nil {
			return nil, err
		}
	} else {
		publisherName, err = syscall.UTF16PtrFromString(name)
		if err != nil {
			return nil, err
		}
	}

	handle, err := _EvtOpenPublisherMetadata(session, publisherName, logFile, 0, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed in EvtOpenPublisherMetadata")
	}

	return &PublisherMetadata{
		Name:   name,
		Handle: handle,
	}, nil
}

func (m *PublisherMetadata) stringProperty(propertyID EvtPublisherMetadataPropertyID) (string, error) {
	v, err := EvtGetPublisherMetadataProperty(m.Handle, propertyID)
	if err != nil {
		return "", err
	}
	switch t := v.(type) {
	case string:
		return t, nil
	case nil:
		return "", nil
	default:
		return "", errors.Errorf("unexpected data type: %T", v)
	}
}

func (m *PublisherMetadata) PublisherGUID() (windows.GUID, error) {
	v, err := EvtGetPublisherMetadataProperty(m.Handle, EvtPublisherMetadataPublisherGuid)
	if err != nil {
		return windows.GUID{}, err
	}
	switch t := v.(type) {
	case windows.GUID:
		return t, nil
	case nil:
		return windows.GUID{}, nil
	default:
		return windows.GUID{}, errors.Errorf("unexpected data type: %T", v)
	}
}

func (m *PublisherMetadata) ResourceFilePath() (string, error) {
	return m.stringProperty(EvtPublisherMetadataResourceFilePath)
}

func (m *PublisherMetadata) ParameterFilePath() (string, error) {
	return m.stringProperty(EvtPublisherMetadataParameterFilePath)
}

func (m *PublisherMetadata) MessageFilePath() (string, error) {
	return m.stringProperty(EvtPublisherMetadataMessageFilePath)
}

func (m *PublisherMetadata) HelpLink() (string, error) {
	return m.stringProperty(EvtPublisherMetadataHelpLink)
}

func (m *PublisherMetadata) PublisherMessageID() (uint32, error) {
	v, err := EvtGetPublisherMetadataProperty(m.Handle, EvtPublisherMetadataPublisherMessageID)
	if err != nil {
		return 0, err
	}
	return v.(uint32), nil
}

func (m *PublisherMetadata) PublisherMessage() (string, error) {
	messageID, err := m.PublisherMessageID()
	if err != nil {
		return "", err
	}
	if int32(messageID) == -1 {
		return "", nil
	}
	return getMessageStringFromMessageID(m, messageID, nil)
}

func (m *PublisherMetadata) Keywords() ([]MetadataKeyword, error) {
	return NewMetadataKeywords(m.Handle)
}

func (m *PublisherMetadata) Opcodes() ([]MetadataOpcode, error) {
	return NewMetadataOpcodes(m.Handle)
}

func (m *PublisherMetadata) Levels() ([]MetadataLevel, error) {
	return NewMetadataLevels(m.Handle)
}

func (m *PublisherMetadata) Tasks() ([]MetadataTask, error) {
	return NewMetadataTasks(m.Handle)
}

func (m *PublisherMetadata) Channels() ([]MetadataChannel, error) {
	return NewMetadataChannels(m.Handle)
}

func (m *PublisherMetadata) EventMetadataIterator() (*EventMetadataIterator, error) {
	return NewEventMetadataIterator(m)
}

type MetadataKeyword struct {
	Name      string
	Mask      uint64
	Message   string
	MessageID uint32
}

func NewMetadataKeywords(publisherMetadataHandle EvtHandle) ([]MetadataKeyword, error) {
	v, err := EvtGetPublisherMetadataProperty(publisherMetadataHandle, EvtPublisherMetadataKeywords)
	if err != nil {
		return nil, err
	}

	arrayHandle, ok := v.(EvtObjectArrayPropertyHandle)
	if !ok {
		return nil, errors.Errorf("unexpected handle type: %T", v)
	}
	defer arrayHandle.Close()

	arrayLen, err := EvtGetObjectArraySize(arrayHandle)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get keyword array length")
	}

	var values []MetadataKeyword
	for i := uint32(0); i < arrayLen; i++ {
		md, err := NewMetadataKeyword(publisherMetadataHandle, arrayHandle, i)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get keyword at array index %v", i)
		}

		values = append(values, *md)
	}

	return values, nil
}

func NewMetadataKeyword(publisherMetadataHandle EvtHandle, arrayHandle EvtObjectArrayPropertyHandle, index uint32) (*MetadataKeyword, error) {
	v, err := EvtGetObjectArrayProperty(arrayHandle, EvtPublisherMetadataKeywordMessageID, index)
	if err != nil {
		return nil, err
	}
	messageID := v.(uint32)

	// The value is -1 if the keyword did not specify a message attribute.
	var message string
	if int32(messageID) != -1 {
		message, err = evtFormatMessage(publisherMetadataHandle, NilHandle, messageID, nil, EvtFormatMessageId)
		if err != nil {
			return nil, err
		}
	}

	v, err = EvtGetObjectArrayProperty(arrayHandle, EvtPublisherMetadataKeywordName, index)
	if err != nil {
		return nil, err
	}
	name := v.(string)

	v, err = EvtGetObjectArrayProperty(arrayHandle, EvtPublisherMetadataKeywordValue, index)
	if err != nil {
		return nil, err
	}
	valueMask := v.(uint64)

	return &MetadataKeyword{
		Name:      name,
		Mask:      valueMask,
		MessageID: messageID,
		Message:   message,
	}, nil
}

type MetadataOpcode struct {
	Name      string
	Mask      uint32
	MessageID uint32
	Message   string
}

func NewMetadataOpcodes(publisherMetadataHandle EvtHandle) ([]MetadataOpcode, error) {
	v, err := EvtGetPublisherMetadataProperty(publisherMetadataHandle, EvtPublisherMetadataOpcodes)
	if err != nil {
		return nil, err
	}

	arrayHandle, ok := v.(EvtObjectArrayPropertyHandle)
	if !ok {
		return nil, errors.Errorf("unexpected handle type: %T", v)
	}
	defer arrayHandle.Close()

	arrayLen, err := EvtGetObjectArraySize(arrayHandle)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get opcode array length")
	}

	var values []MetadataOpcode
	for i := uint32(0); i < arrayLen; i++ {
		md, err := NewMetadataOpcode(publisherMetadataHandle, arrayHandle, i)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get opcode at array index %v", i)
		}

		values = append(values, *md)
	}

	return values, nil
}

func NewMetadataOpcode(publisherMetadataHandle EvtHandle, arrayHandle EvtObjectArrayPropertyHandle, index uint32) (*MetadataOpcode, error) {
	v, err := EvtGetObjectArrayProperty(arrayHandle, EvtPublisherMetadataOpcodeMessageID, index)
	if err != nil {
		return nil, err
	}
	messageID := v.(uint32)

	// The value is -1 if the opcode did not specify a message attribute.
	var message string
	if int32(messageID) != -1 {
		message, err = evtFormatMessage(publisherMetadataHandle, NilHandle, messageID, nil, EvtFormatMessageId)
		if err != nil {
			return nil, err
		}
	}

	v, err = EvtGetObjectArrayProperty(arrayHandle, EvtPublisherMetadataOpcodeName, index)
	if err != nil {
		return nil, err
	}
	name := v.(string)

	v, err = EvtGetObjectArrayProperty(arrayHandle, EvtPublisherMetadataOpcodeValue, index)
	if err != nil {
		return nil, err
	}
	valueMask := v.(uint32)

	return &MetadataOpcode{
		Name:      name,
		Mask:      valueMask,
		MessageID: messageID,
		Message:   message,
	}, nil
}

type MetadataLevel struct {
	Name      string
	Mask      uint32
	MessageID uint32
	Message   string
}

func NewMetadataLevels(publisherMetadataHandle EvtHandle) ([]MetadataLevel, error) {
	v, err := EvtGetPublisherMetadataProperty(publisherMetadataHandle, EvtPublisherMetadataLevels)
	if err != nil {
		return nil, err
	}

	arrayHandle, ok := v.(EvtObjectArrayPropertyHandle)
	if !ok {
		return nil, errors.Errorf("unexpected handle type: %T", v)
	}
	defer arrayHandle.Close()

	arrayLen, err := EvtGetObjectArraySize(arrayHandle)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get level array length")
	}

	var values []MetadataLevel
	for i := uint32(0); i < arrayLen; i++ {
		md, err := NewMetadataLevel(publisherMetadataHandle, arrayHandle, i)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get level at array index %v", i)
		}

		values = append(values, *md)
	}

	return values, nil
}

func NewMetadataLevel(publisherMetadataHandle EvtHandle, arrayHandle EvtObjectArrayPropertyHandle, index uint32) (*MetadataLevel, error) {
	v, err := EvtGetObjectArrayProperty(arrayHandle, EvtPublisherMetadataLevelMessageID, index)
	if err != nil {
		return nil, err
	}
	messageID := v.(uint32)

	// The value is -1 if the level did not specify a message attribute.
	var message string
	if int32(messageID) != -1 {
		message, err = evtFormatMessage(publisherMetadataHandle, NilHandle, messageID, nil, EvtFormatMessageId)
		if err != nil {
			return nil, err
		}
	}

	v, err = EvtGetObjectArrayProperty(arrayHandle, EvtPublisherMetadataLevelName, index)
	if err != nil {
		return nil, err
	}
	name := v.(string)

	v, err = EvtGetObjectArrayProperty(arrayHandle, EvtPublisherMetadataLevelValue, index)
	if err != nil {
		return nil, err
	}
	valueMask := v.(uint32)

	return &MetadataLevel{
		Name:      name,
		Mask:      valueMask,
		MessageID: messageID,
		Message:   message,
	}, nil
}

type MetadataTask struct {
	Name      string
	Mask      uint32
	MessageID uint32
	Message   string
	EventGUID windows.GUID
}

func NewMetadataTasks(publisherMetadataHandle EvtHandle) ([]MetadataTask, error) {
	v, err := EvtGetPublisherMetadataProperty(publisherMetadataHandle, EvtPublisherMetadataTasks)
	if err != nil {
		return nil, err
	}

	arrayHandle, ok := v.(EvtObjectArrayPropertyHandle)
	if !ok {
		return nil, errors.Errorf("unexpected handle type: %T", v)
	}
	defer arrayHandle.Close()

	arrayLen, err := EvtGetObjectArraySize(arrayHandle)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get task array length")
	}

	var values []MetadataTask
	for i := uint32(0); i < arrayLen; i++ {
		md, err := NewMetadataTask(publisherMetadataHandle, arrayHandle, i)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get task at array index %v", i)
		}

		values = append(values, *md)
	}

	return values, nil
}

func NewMetadataTask(publisherMetadataHandle EvtHandle, arrayHandle EvtObjectArrayPropertyHandle, index uint32) (*MetadataTask, error) {
	v, err := EvtGetObjectArrayProperty(arrayHandle, EvtPublisherMetadataTaskMessageID, index)
	if err != nil {
		return nil, err
	}
	messageID := v.(uint32)

	// The value is -1 if the task did not specify a message attribute.
	var message string
	if int32(messageID) != -1 {
		message, err = evtFormatMessage(publisherMetadataHandle, NilHandle, messageID, nil, EvtFormatMessageId)
		if err != nil {
			return nil, err
		}
	}

	v, err = EvtGetObjectArrayProperty(arrayHandle, EvtPublisherMetadataTaskName, index)
	if err != nil {
		return nil, err
	}
	name := v.(string)

	v, err = EvtGetObjectArrayProperty(arrayHandle, EvtPublisherMetadataTaskValue, index)
	if err != nil {
		return nil, err
	}
	valueMask := v.(uint32)

	v, err = EvtGetObjectArrayProperty(arrayHandle, EvtPublisherMetadataTaskEventGuid, index)
	if err != nil {
		return nil, err
	}
	guid := v.(windows.GUID)

	return &MetadataTask{
		Name:      name,
		Mask:      valueMask,
		MessageID: messageID,
		Message:   message,
		EventGUID: guid,
	}, nil
}

type MetadataChannel struct {
	Name      string
	Index     uint32
	ID        uint32
	Message   string
	MessageID uint32
}

func NewMetadataChannels(publisherMetadataHandle EvtHandle) ([]MetadataChannel, error) {
	v, err := EvtGetPublisherMetadataProperty(publisherMetadataHandle, EvtPublisherMetadataChannelReferences)
	if err != nil {
		return nil, err
	}

	arrayHandle, ok := v.(EvtObjectArrayPropertyHandle)
	if !ok {
		return nil, errors.Errorf("unexpected handle type: %T", v)
	}
	defer arrayHandle.Close()

	arrayLen, err := EvtGetObjectArraySize(arrayHandle)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get task array length")
	}

	var values []MetadataChannel
	for i := uint32(0); i < arrayLen; i++ {
		md, err := NewMetadataChannel(publisherMetadataHandle, arrayHandle, i)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get task at array index %v", i)
		}

		values = append(values, *md)
	}

	return values, nil
}

func NewMetadataChannel(publisherMetadataHandle EvtHandle, arrayHandle EvtObjectArrayPropertyHandle, index uint32) (*MetadataChannel, error) {
	v, err := EvtGetObjectArrayProperty(arrayHandle, EvtPublisherMetadataChannelReferenceMessageID, index)
	if err != nil {
		return nil, err
	}
	messageID := v.(uint32)

	// The value is -1 if the task did not specify a message attribute.
	var message string
	if int32(messageID) != -1 {
		message, err = evtFormatMessage(publisherMetadataHandle, NilHandle, messageID, nil, EvtFormatMessageId)
		if err != nil {
			return nil, err
		}
	}

	// Channel name.
	v, err = EvtGetObjectArrayProperty(arrayHandle, EvtPublisherMetadataChannelReferencePath, index)
	if err != nil {
		return nil, err
	}
	name := v.(string)

	v, err = EvtGetObjectArrayProperty(arrayHandle, EvtPublisherMetadataChannelReferenceIndex, index)
	if err != nil {
		return nil, err
	}
	channelIndex := v.(uint32)

	v, err = EvtGetObjectArrayProperty(arrayHandle, EvtPublisherMetadataChannelReferenceID, index)
	if err != nil {
		return nil, err
	}
	id := v.(uint32)

	return &MetadataChannel{
		Name:      name,
		Index:     channelIndex,
		ID:        id,
		MessageID: messageID,
		Message:   message,
	}, nil
}

type EventMetadataIterator struct {
	Publisher               *PublisherMetadata
	eventMetadataEnumHandle EvtHandle
	currentEvent            EvtHandle
	lastErr                 error
}

func NewEventMetadataIterator(publisher *PublisherMetadata) (*EventMetadataIterator, error) {
	eventMetadataEnumHandle, err := _EvtOpenEventMetadataEnum(publisher.Handle, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open event metadata enumerator with EvtOpenEventMetadataEnum")
	}

	return &EventMetadataIterator{
		Publisher:               publisher,
		eventMetadataEnumHandle: eventMetadataEnumHandle,
	}, nil
}

func (itr *EventMetadataIterator) Close() error {
	return multierr.Combine(
		_EvtClose(itr.eventMetadataEnumHandle),
		_EvtClose(itr.currentEvent),
	)
}

// Next advances to the next event handle. It returns false when there are
// no more items or an error occurred. You should call Err() to check for an
// error.
func (itr *EventMetadataIterator) Next() bool {
	// Close existing handle.
	itr.currentEvent.Close()

	var err error
	itr.currentEvent, err = _EvtNextEventMetadata(itr.eventMetadataEnumHandle, 0)
	if err != nil {
		if err != windows.ERROR_NO_MORE_ITEMS {
			itr.lastErr = errors.Wrap(err, "failed advancing to next event metadata handle")
		}
		return false
	}
	return true
}

// Err returns an error if Next() failed due to an error.
func (itr *EventMetadataIterator) Err() error {
	return itr.lastErr
}

func typeCastError(expected, got interface{}) error {
	return errors.Errorf("wrong type for property. expected:%T got:%T", expected, got)
}

func (itr *EventMetadataIterator) uint32Property(propertyID EvtEventMetadataPropertyID) (uint32, error) {
	v, err := GetEventMetadataProperty(itr.currentEvent, propertyID)
	if err != nil {
		return 0, err
	}
	value, ok := v.(uint32)
	if !ok {
		return value, typeCastError(value, v)
	}
	return value, nil
}

func (itr *EventMetadataIterator) uint64Property(propertyID EvtEventMetadataPropertyID) (uint64, error) {
	v, err := GetEventMetadataProperty(itr.currentEvent, propertyID)
	if err != nil {
		return 0, err
	}
	value, ok := v.(uint64)
	if !ok {
		return value, typeCastError(value, v)
	}
	return value, nil
}

func (itr *EventMetadataIterator) stringProperty(propertyID EvtEventMetadataPropertyID) (string, error) {
	v, err := GetEventMetadataProperty(itr.currentEvent, propertyID)
	if err != nil {
		return "", err
	}
	value, ok := v.(string)
	if !ok {
		return value, typeCastError(value, v)
	}
	return value, nil
}

func (itr *EventMetadataIterator) EventID() (uint32, error) {
	return itr.uint32Property(EventMetadataEventID)
}

func (itr *EventMetadataIterator) Version() (uint32, error) {
	return itr.uint32Property(EventMetadataEventVersion)
}

func (itr *EventMetadataIterator) Channel() (uint32, error) {
	return itr.uint32Property(EventMetadataEventVersion)
}

func (itr *EventMetadataIterator) Level() (uint32, error) {
	return itr.uint32Property(EventMetadataEventLevel)
}

func (itr *EventMetadataIterator) Opcode() (uint32, error) {
	return itr.uint32Property(EventMetadataEventOpcode)
}

func (itr *EventMetadataIterator) Task() (uint32, error) {
	return itr.uint32Property(EventMetadataEventTask)
}

func (itr *EventMetadataIterator) Keyword() (uint64, error) {
	return itr.uint64Property(EventMetadataEventKeyword)
}

func (itr *EventMetadataIterator) MessageID() (uint32, error) {
	return itr.uint32Property(EventMetadataEventMessageID)
}

func (itr *EventMetadataIterator) Template() (string, error) {
	return itr.stringProperty(EventMetadataEventTemplate)
}

// Message returns the raw event description without doing any substitutions
// (e.g. the message will contain %1, %2, etc. as parameter placeholders).
func (itr *EventMetadataIterator) Message() (string, error) {
	messageID, err := itr.MessageID()
	if err != nil {
		return "", err
	}
	// If the event definition does not specify a message, the value is –1.
	if int32(messageID) == -1 {
		return "", nil
	}

	return getMessageStringFromMessageID(itr.Publisher, messageID, nil)
}
