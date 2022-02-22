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
	"encoding/binary"
	"fmt"
	"strconv"
	"sync"
	"text/template"
	"time"
	"unsafe"

	"github.com/cespare/xxhash/v2"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/sys/windows"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/winlogbeat/sys"
	"github.com/elastic/beats/v7/winlogbeat/sys/winevent"
)

const (
	// keywordClassic indicates the log was published with the "classic" event
	// logging API.
	// https://docs.microsoft.com/en-us/dotnet/api/system.diagnostics.eventing.reader.standardeventkeywords?view=netframework-4.8
	keywordClassic = 0x80000000000000
)

// Renderer is used for converting event log handles into complete events.
type Renderer struct {
	// Cache of publisher metadata. Maps publisher names to stored metadata.
	metadataCache map[string]*PublisherMetadataStore
	// Mutex to guard the metadataCache. The other members are immutable.
	mutex sync.RWMutex

	session       EvtHandle // Session handle if working with remote log.
	systemContext EvtHandle // Render context for system values.
	userContext   EvtHandle // Render context for user values (event data).
	log           *logp.Logger
}

// NewRenderer returns a new Renderer.
func NewRenderer(session EvtHandle, log *logp.Logger) (*Renderer, error) {
	systemContext, err := _EvtCreateRenderContext(0, 0, EvtRenderContextSystem)
	if err != nil {
		return nil, errors.Wrap(err, "failed in EvtCreateRenderContext for system context")
	}

	userContext, err := _EvtCreateRenderContext(0, 0, EvtRenderContextUser)
	if err != nil {
		return nil, errors.Wrap(err, "failed in EvtCreateRenderContext for user context")
	}

	return &Renderer{
		metadataCache: map[string]*PublisherMetadataStore{},
		session:       session,
		systemContext: systemContext,
		userContext:   userContext,
		log:           log.Named("renderer"),
	}, nil
}

// Close closes all handles held by the Renderer.
func (r *Renderer) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	errs := []error{r.systemContext.Close(), r.userContext.Close()}
	for _, md := range r.metadataCache {
		if err := md.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return multierr.Combine(errs...)
}

// Render renders the event handle into an Event.
func (r *Renderer) Render(handle EvtHandle) (*winevent.Event, error) {
	event := &winevent.Event{}

	if err := r.renderSystem(handle, event); err != nil {
		return nil, errors.Wrap(err, "failed to render system properties")
	}

	// From this point on it will return both the event and any errors. It's
	// critical to not drop data.
	var errs []error

	// This always returns a non-nil value (even on error).
	md, err := r.getPublisherMetadata(event.Provider.Name)
	if err != nil {
		errs = append(errs, err)
	}

	// Associate raw system properties to names (e.g. level=2 to Error).
	winevent.EnrichRawValuesWithNames(&md.WinMeta, event)

	eventData, fingerprint, err := r.renderUser(handle, event)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "failed to render event data"))
	}

	// Load cached event metadata or try to bootstrap it from the event's XML.
	eventMeta := md.getEventMetadata(uint16(event.EventIdentifier.ID), fingerprint, handle)

	// Associate key names with the event data values.
	r.addEventData(eventMeta, eventData, event)

	if event.Message, err = r.formatMessage(md, eventMeta, handle, eventData, uint16(event.EventIdentifier.ID)); err != nil {
		errs = append(errs, errors.Wrap(err, "failed to get the event message string"))
	}

	if len(errs) > 0 {
		return event, multierr.Combine(errs...)
	}
	return event, nil
}

// getPublisherMetadata return a PublisherMetadataStore for the provider. It
// never returns nil, but may return an error if it couldn't open a publisher.
func (r *Renderer) getPublisherMetadata(publisher string) (*PublisherMetadataStore, error) {
	var err error

	// NOTE: This code uses double-check locking to elevate to a write-lock
	// when a cache value needs initialized.
	r.mutex.RLock()

	// Lookup cached value.
	md, found := r.metadataCache[publisher]
	if !found {
		// Elevate to write lock.
		r.mutex.RUnlock()
		r.mutex.Lock()
		defer r.mutex.Unlock()

		// Double-check if the condition changed while upgrading the lock.
		md, found = r.metadataCache[publisher]
		if found {
			return md, nil
		}

		// Load metadata from the publisher.
		md, err = NewPublisherMetadataStore(r.session, publisher, r.log)
		if err != nil {
			// Return an empty store on error (can happen in cases where the
			// log was forwarded and the provider doesn't exist on collector).
			md = NewEmptyPublisherMetadataStore(publisher, r.log)
			err = errors.Wrapf(err, "failed to load publisher metadata for %v "+
				"(returning an empty metadata store)", publisher)
		}
		r.metadataCache[publisher] = md
	} else {
		r.mutex.RUnlock()
	}

	return md, err
}

// renderSystem writes all the system context properties into the event.
func (r *Renderer) renderSystem(handle EvtHandle, event *winevent.Event) error {
	bb, propertyCount, err := r.render(r.systemContext, handle)
	if err != nil {
		return errors.Wrap(err, "failed to get system values")
	}
	defer bb.Free()

	for i := 0; i < int(propertyCount); i++ {
		property := EvtSystemPropertyID(i)
		offset := i * int(sizeofEvtVariant)
		evtVar := (*EvtVariant)(unsafe.Pointer(bb.PtrAt(offset)))

		data, err := evtVar.Data(bb.Bytes())
		if err != nil || data == nil {
			continue
		}

		switch property {
		case EvtSystemProviderName:
			event.Provider.Name = data.(string)
		case EvtSystemProviderGuid:
			event.Provider.GUID = data.(windows.GUID).String()
		case EvtSystemEventID:
			event.EventIdentifier.ID = uint32(data.(uint16))
		case EvtSystemQualifiers:
			event.EventIdentifier.Qualifiers = data.(uint16)
		case EvtSystemLevel:
			event.LevelRaw = data.(uint8)
		case EvtSystemTask:
			event.TaskRaw = data.(uint16)
		case EvtSystemOpcode:
			event.OpcodeRaw = data.(uint8)
		case EvtSystemKeywords:
			event.KeywordsRaw = winevent.HexInt64(data.(hexInt64))
		case EvtSystemTimeCreated:
			event.TimeCreated.SystemTime = data.(time.Time)
		case EvtSystemEventRecordId:
			event.RecordID = data.(uint64)
		case EvtSystemActivityID:
			event.Correlation.ActivityID = data.(windows.GUID).String()
		case EvtSystemRelatedActivityID:
			event.Correlation.RelatedActivityID = data.(windows.GUID).String()
		case EvtSystemProcessID:
			event.Execution.ProcessID = data.(uint32)
		case EvtSystemThreadID:
			event.Execution.ThreadID = data.(uint32)
		case EvtSystemChannel:
			event.Channel = data.(string)
		case EvtSystemComputer:
			event.Computer = data.(string)
		case EvtSystemUserID:
			sid := data.(*windows.SID)
			event.User.Identifier = sid.String()
			var accountType uint32
			event.User.Name, event.User.Domain, accountType, _ = sid.LookupAccount("")
			event.User.Type = winevent.SIDType(accountType)
		case EvtSystemVersion:
			event.Version = winevent.Version(data.(uint8))
		}
	}

	return nil
}

// renderUser returns the event/user data values. This does not provide the
// parameter names. It computes a fingerprint of the values types to help the
// caller match the correct names to the returned values.
func (r *Renderer) renderUser(handle EvtHandle, event *winevent.Event) (values []interface{}, fingerprint uint64, err error) {
	bb, propertyCount, err := r.render(r.userContext, handle)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to get user values")
	}
	defer bb.Free()

	if propertyCount == 0 {
		return nil, 0, nil
	}

	// Fingerprint the argument types to help ensure we match these values with
	// the correct event data parameter names.
	argumentHash := xxhash.New()
	binary.Write(argumentHash, binary.LittleEndian, propertyCount)

	values = make([]interface{}, propertyCount)
	for i := 0; i < propertyCount; i++ {
		offset := i * int(sizeofEvtVariant)
		evtVar := (*EvtVariant)(unsafe.Pointer(bb.PtrAt(offset)))
		binary.Write(argumentHash, binary.LittleEndian, uint32(evtVar.Type))

		values[i], err = evtVar.Data(bb.Bytes())
		if err != nil {
			r.log.Warnw("Failed to read event/user data value. Using nil.",
				"provider", event.Provider.Name,
				"event_id", event.EventIdentifier.ID,
				"value_index", i,
				"value_type", evtVar.Type.String(),
				"error", err,
			)
		}
	}

	return values, argumentHash.Sum64(), nil
}

// render uses EvtRender to event data. The caller must free() the returned when
// done accessing the bytes.
func (r *Renderer) render(context EvtHandle, eventHandle EvtHandle) (*sys.PooledByteBuffer, int, error) {
	var bufferUsed, propertyCount uint32

	err := _EvtRender(context, eventHandle, EvtRenderEventValues, 0, nil, &bufferUsed, &propertyCount)
	if err != nil && err != windows.ERROR_INSUFFICIENT_BUFFER {
		return nil, 0, errors.Wrap(err, "failed in EvtRender")
	}

	if propertyCount == 0 {
		return nil, 0, nil
	}

	bb := sys.NewPooledByteBuffer()
	bb.Reserve(int(bufferUsed))

	err = _EvtRender(context, eventHandle, EvtRenderEventValues, uint32(bb.Len()), bb.PtrAt(0), &bufferUsed, &propertyCount)
	if err != nil {
		bb.Free()
		return nil, 0, errors.Wrap(err, "failed in EvtRender")
	}

	return bb, int(propertyCount), nil
}

// addEventData adds the event/user data values to the event.
func (r *Renderer) addEventData(evtMeta *EventMetadata, values []interface{}, event *winevent.Event) {
	if len(values) == 0 {
		return
	}

	if evtMeta == nil {
		r.log.Warnw("Event metadata not found.",
			"provider", event.Provider.Name,
			"event_id", event.EventIdentifier.ID)
	} else if len(values) != len(evtMeta.EventData) {
		r.log.Warnw("The number of event data parameters doesn't match the number "+
			"of parameters in the template.",
			"provider", event.Provider.Name,
			"event_id", event.EventIdentifier.ID,
			"event_parameter_count", len(values),
			"template_parameter_count", len(evtMeta.EventData),
			"template_version", evtMeta.Version,
			"event_version", event.Version)
	}

	// Fallback to paramN naming when the value does not exist in event data.
	// This can happen for legacy providers without manifests. This can also
	// happen if the installed provider manifest doesn't match the version that
	// produced the event (forwarded events, reading from evtx, or software was
	// updated). If software was updated it could also be that this cached
	// template is now stale.
	paramName := func(idx int) string {
		if evtMeta != nil && idx < len(evtMeta.EventData) {
			return evtMeta.EventData[idx].Name
		}
		return "param" + strconv.Itoa(idx)
	}

	for i, v := range values {
		var strVal string
		switch t := v.(type) {
		case string:
			strVal = t
		case *windows.SID:
			strVal = t.String()
		default:
			strVal = fmt.Sprintf("%v", v)
		}

		event.EventData.Pairs = append(event.EventData.Pairs, winevent.KeyValue{
			Key:   paramName(i),
			Value: strVal,
		})
	}

	return
}

// formatMessage adds the message to the event.
func (r *Renderer) formatMessage(publisherMeta *PublisherMetadataStore,
	eventMeta *EventMetadata, eventHandle EvtHandle, values []interface{},
	eventID uint16) (string, error,
) {
	if eventMeta != nil {
		if eventMeta.MsgStatic != "" {
			return eventMeta.MsgStatic, nil
		} else if eventMeta.MsgTemplate != nil {
			return r.formatMessageFromTemplate(eventMeta.MsgTemplate, values)
		}
	}

	// Fallback to the trying EvtFormatMessage mechanism.
	// This is the path for forwarded events in RenderedText mode where the
	// local publisher metadata is not present. NOTE that if the local publisher
	// metadata exists it will be preferred over the RenderedText. A config
	// option might be desirable to control this behavior.
	r.log.Debugf("Falling back to EvtFormatMessage for event ID %d.", eventID)
	return getMessageString(publisherMeta.Metadata, eventHandle, 0, nil)
}

// formatMessageFromTemplate creates the message by executing the stored Go
// text/template with the event/user data values.
func (r *Renderer) formatMessageFromTemplate(msgTmpl *template.Template, values []interface{}) (string, error) {
	bb := sys.NewPooledByteBuffer()
	defer bb.Free()

	if err := msgTmpl.Execute(bb, values); err != nil {
		return "", errors.Wrapf(err, "failed to execute template with data=%#v template=%v", values, msgTmpl.Root.String())
	}

	return string(bb.Bytes()), nil
}
