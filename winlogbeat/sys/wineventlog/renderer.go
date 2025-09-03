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

package wineventlog

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"text/template"
	"time"
	"unsafe"

	"github.com/cespare/xxhash/v2"
	"go.uber.org/multierr"
	"golang.org/x/sys/windows"

	"github.com/elastic/beats/v7/winlogbeat/sys"
	"github.com/elastic/beats/v7/winlogbeat/sys/winevent"
	"github.com/elastic/elastic-agent-libs/logp"
)

<<<<<<< HEAD
// Renderer is used for converting event log handles into complete events.
type Renderer struct {
	// Mutex to guard the metadataCache. The other members are immutable.
	mutex sync.RWMutex
	// Cache of publisher metadata. Maps publisher names to stored metadata.
	metadataCache map[string]*PublisherMetadataStore

	session       EvtHandle // Session handle if working with remote log.
=======
type EventRenderer interface {
	Render(handle EvtHandle) (event *winevent.Event, xml string, err error)
	Close() error
}

// Renderer is used for converting event log handles into complete events.
type Renderer struct {
	metadataCache *publisherMetadataCache
>>>>>>> abcb373d9 ([winlogbeat] Fix forwarded event handling and add channel error resilience (#46190))
	systemContext EvtHandle // Render context for system values.
	userContext   EvtHandle // Render context for user values (event data).
	log           *logp.Logger
}

// NewRenderer returns a new Renderer.
<<<<<<< HEAD
func NewRenderer(session EvtHandle, log *logp.Logger) (*Renderer, error) {
=======
func NewRenderer(locale uint32, session EvtHandle, log *logp.Logger) (*Renderer, error) {
>>>>>>> abcb373d9 ([winlogbeat] Fix forwarded event handling and add channel error resilience (#46190))
	systemContext, err := _EvtCreateRenderContext(0, nil, EvtRenderContextSystem)
	if err != nil {
		return nil, fmt.Errorf("failed in EvtCreateRenderContext for system context: %w", err)
	}

	userContext, err := _EvtCreateRenderContext(0, nil, EvtRenderContextUser)
	if err != nil {
		return nil, fmt.Errorf("failed in EvtCreateRenderContext for user context: %w", err)
	}

	return &Renderer{
<<<<<<< HEAD
		metadataCache: map[string]*PublisherMetadataStore{},
		session:       session,
=======
		metadataCache: newPublisherMetadataCache(session, locale, rlog),
>>>>>>> abcb373d9 ([winlogbeat] Fix forwarded event handling and add channel error resilience (#46190))
		systemContext: systemContext,
		userContext:   userContext,
		log:           log.Named("renderer"),
	}, nil
}

// Close closes all handles held by the Renderer.
func (r *Renderer) Close() error {
	if r == nil {
		return errors.New("closing nil renderer")
	}
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
		return nil, fmt.Errorf("failed to render system properties: %w", err)
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
<<<<<<< HEAD
	winevent.EnrichRawValuesWithNames(&md.WinMeta, event)
=======
	enrichRawValuesWithNames(&md.WinMeta, event)
>>>>>>> abcb373d9 ([winlogbeat] Fix forwarded event handling and add channel error resilience (#46190))

	eventData, fingerprint, err := r.renderUser(handle, event)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to render event data: %w", err))
	}

	// Load cached event metadata or try to bootstrap it from the event's XML.
	eventMeta := md.getEventMetadata(uint16(event.EventIdentifier.ID), fingerprint, handle)

	// Associate key names with the event data values.
	r.addEventData(eventMeta, eventData, event)

	if event.Message, err = r.formatMessage(md, eventMeta, handle, eventData, uint16(event.EventIdentifier.ID)); err != nil {
		errs = append(errs, fmt.Errorf("failed to get the event message string: %w", err))
	}

<<<<<<< HEAD
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
			err = fmt.Errorf("failed to load publisher metadata for %v "+
				"(returning an empty metadata store): %w", publisher, err)
		}
		r.metadataCache[publisher] = md
	} else {
		r.mutex.RUnlock()
	}

	return md, err
=======
	return event, "", multierr.Combine(errs...)
>>>>>>> abcb373d9 ([winlogbeat] Fix forwarded event handling and add channel error resilience (#46190))
}

// renderSystem writes all the system context properties into the event.
func (r *Renderer) renderSystem(handle EvtHandle, event *winevent.Event) error {
	bb, propertyCount, err := r.render(r.systemContext, handle)
	if err != nil {
		return fmt.Errorf("failed to get system values: %w", err)
	}
	defer bb.Free()

	for i := 0; i < propertyCount; i++ {
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
			if event.OpcodeRaw == nil {
				event.OpcodeRaw = new(uint8)
			}
			*event.OpcodeRaw = data.(uint8)
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
		return nil, 0, fmt.Errorf("failed to get user values: %w", err)
	}
	defer bb.Free()

	if propertyCount == 0 {
		return nil, 0, nil
	}

	// Fingerprint the argument types to help ensure we match these values with
	// the correct event data parameter names.
	argumentHash := xxhash.New()
	err = binary.Write(argumentHash, binary.LittleEndian, int64(propertyCount))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to hash property count: %w", err)
	}

	values = make([]interface{}, propertyCount)
	for i := 0; i < propertyCount; i++ {
		offset := i * int(sizeofEvtVariant)
		evtVar := (*EvtVariant)(unsafe.Pointer(bb.PtrAt(offset)))
		binary.Write(argumentHash, binary.LittleEndian, uint32(evtVar.Type)) //nolint:errcheck // Hash writes never fail.

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
	if err != nil && err != windows.ERROR_INSUFFICIENT_BUFFER { //nolint:errorlint // This is an errno or nil.
		return nil, 0, fmt.Errorf("failed in EvtRender: %w", err)
	}

	if propertyCount == 0 {
		return nil, 0, nil
	}

	bb := sys.NewPooledByteBuffer()
	bb.Reserve(int(bufferUsed))

	err = _EvtRender(context, eventHandle, EvtRenderEventValues, uint32(bb.Len()), bb.PtrAt(0), &bufferUsed, &propertyCount)
	if err != nil {
		bb.Free()
		return nil, 0, fmt.Errorf("failed in EvtRender: %w", err)
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
<<<<<<< HEAD
	return getMessageString(publisherMeta.Metadata, eventHandle, 0, nil)
=======
	metadata := publisherMeta
	return getMessageString(metadata, eventHandle, 0, nil)
>>>>>>> abcb373d9 ([winlogbeat] Fix forwarded event handling and add channel error resilience (#46190))
}

// formatMessageFromTemplate creates the message by executing the stored Go
// text/template with the event/user data values.
func (r *Renderer) formatMessageFromTemplate(msgTmpl *template.Template, values []interface{}) (string, error) {
	bb := sys.NewPooledByteBuffer()
	defer bb.Free()

	if err := msgTmpl.Execute(bb, values); err != nil {
		return "", fmt.Errorf("failed to execute template with data=%#v template=%v: %w", values, msgTmpl.Root.String(), err)
	}

	return string(bb.Bytes()), nil
}
<<<<<<< HEAD
=======

// XMLRenderer is used for converting event log handles into complete events.
type XMLRenderer struct {
	isForwarded   bool
	metadataCache *publisherMetadataCache
	renderBuf     []byte
	outBuf        *sys.ByteBuffer

	render func(event EvtHandle, out io.Writer) error // Function for rendering the event to XML.

	log *logp.Logger
}

// NewXMLRenderer returns a new Renderer.
func NewXMLRenderer(locale uint32, isForwarded bool, session EvtHandle, log *logp.Logger) *XMLRenderer {
	const renderBufferSize = 1 << 19 // 512KB, 256K wide characters
	rlog := log.Named("xml_renderer")
	r := &XMLRenderer{
		isForwarded:   isForwarded,
		renderBuf:     make([]byte, renderBufferSize),
		outBuf:        sys.NewByteBuffer(renderBufferSize),
		metadataCache: newPublisherMetadataCache(session, locale, rlog),
		log:           rlog,
	}
	// Forwarded events should be rendered using RenderEventXML. It is more
	// efficient and does not attempt to use local message files for rendering
	// the event's message.
	switch isForwarded {
	case true:
		r.render = func(event EvtHandle, out io.Writer) error {
			return RenderEventXML(event, r.renderBuf, out)
		}
	case false:
		r.render = func(event EvtHandle, out io.Writer) error {
			get := func(providerName string) EvtHandle {
				md, _ := r.metadataCache.getPublisherStore(providerName)
				if md.Metadata != nil {
					return md.Metadata.Handle
				}
				return NilHandle
			}
			return RenderEvent(event, locale, r.renderBuf, get, out)
		}
	}
	return r
}

// Close closes all handles held by the Renderer.
func (r *XMLRenderer) Close() error {
	if r == nil {
		return errors.New("closing nil renderer")
	}
	return r.metadataCache.close()
}

// Render renders the event handle into an Event.
func (r *XMLRenderer) Render(handle EvtHandle) (*winevent.Event, string, error) {
	// From this point on it will return both the event and any errors. It's
	// critical to not drop data.
	var errs []error

	r.outBuf.Reset()
	err := r.render(handle, r.outBuf)
	if err != nil {
		errs = append(errs, err)
	}
	outBytes := r.outBuf.Bytes()
	event := r.buildEventFromXML(outBytes, err)

	// For forwarded events, avoid publisher metadata cache to prevent pollution
	// and version mismatches. Use static enrichment only.
	if r.isForwarded {
		enrichRawValuesWithNames(nil, event)
		return event, string(outBytes), multierr.Combine(errs...)
	}

	// This always returns a non-nil value (even on error).
	md, err := r.metadataCache.getPublisherStore(event.Provider.Name)
	if err != nil {
		errs = append(errs, err)
	}

	// Associate raw system properties to names (e.g. level=2 to Error).
	enrichRawValuesWithNames(&md.WinMeta, event)

	if event.Message == "" {
		if event.Message, err = getMessageString(md.Metadata, handle, 0, nil); err != nil {
			errs = append(errs, fmt.Errorf("failed to get the event message string: %w", err))
		}
	}

	var pairs *[]winevent.KeyValue
	if len(event.UserData.Pairs) > 0 {
		pairs = &event.UserData.Pairs
	} else if len(event.EventData.Pairs) > 0 {
		pairs = &event.EventData.Pairs
	}

	if pairs != nil {
		for i, pair := range *pairs {
			(*pairs)[i].Value = expandMessageIDs(md, pair.Value)
		}
	}

	return event, string(outBytes), multierr.Combine(errs...)
}

func (r *XMLRenderer) buildEventFromXML(x []byte, recoveredErr error) *winevent.Event {
	e, err := winevent.UnmarshalXML(x)
	if err != nil {
		e.RenderErr = append(e.RenderErr, err.Error())
	}

	err = winevent.PopulateAccount(&e.User)
	if err != nil {
		r.log.Debugf("SID %s account lookup failed. %v",
			e.User.Identifier, err)
	}

	if e.RenderErrorCode != 0 {
		// Convert the render error code to an error message that can be
		// included in the "error.message" field.
		e.RenderErr = append(e.RenderErr, syscall.Errno(e.RenderErrorCode).Error())
	} else if recoveredErr != nil {
		e.RenderErr = append(e.RenderErr, recoveredErr.Error())
	}

	return &e
}

func enrichRawValuesWithNames(m *winevent.WinMeta, event *winevent.Event) {
	winevent.EnrichRawValuesWithNames(m, event)
	if event.Level == "" {
		// Fallback on LevelRaw if the Level is not set in the RenderingInfo.
		event.Level = EventLevel(event.LevelRaw).String()
	}
}
>>>>>>> abcb373d9 ([winlogbeat] Fix forwarded event handling and add channel error resilience (#46190))
