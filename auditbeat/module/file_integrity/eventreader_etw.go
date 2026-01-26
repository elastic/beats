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

package file_integrity

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/sys/windows"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/reader/etw"
	"github.com/elastic/elastic-agent-libs/logp"
)

// ETW keyword constants for enabling specific event categories
const (
	// File system event keywords
	KERNEL_FILE_KEYWORD_FILENAME            = 0x10  // File name operations
	KERNEL_FILE_KEYWORD_FILEIO              = 0x20  // File I/O operations
	KERNEL_FILE_KEYWORD_DELETE_PATH         = 0x400 // File deletion events
	KERNEL_FILE_KEYWORD_RENAME_SETLINK_PATH = 0x800 // File rename/link events

	// Combined file keywords for comprehensive file monitoring
	fileKeywords = KERNEL_FILE_KEYWORD_FILENAME |
		KERNEL_FILE_KEYWORD_FILEIO |
		KERNEL_FILE_KEYWORD_DELETE_PATH |
		KERNEL_FILE_KEYWORD_RENAME_SETLINK_PATH

	// Process event keywords
	WINEVENT_KEYWORD_PROCESS = 0x10 // Process lifecycle events
)

// ETW event IDs for file system operations
const (
	fileNameCreate     = 10 // File name creation event
	fileCreate         = 12 // File handle creation
	fileClose          = 14 // File handle close
	fileWrite          = 16 // File write operation
	fileSetInformation = 17 // File attribute/metadata changes
	fileDeletePath     = 26 // File deletion
	fileRenamePath     = 27 // File rename operation
	fileSetLinkPath    = 28 // Hard link creation
	fileSetSecurity    = 31 // File security descriptor changes
	fileSetEA          = 33 // Extended attributes modification

	// Process lifecycle events
	processStart = 1 // Process creation
	processStop  = 2 // Process termination
)

var (
	// Microsoft-Windows-Kernel-File provider GUID: "{edd08927-9cc4-4e65-b970-c2560fb5c289}"
	// This provider generates file system related events
	kernelFileProviderGUID = windows.GUID{
		Data1: 0xedd08927,
		Data2: 0x9cc4,
		Data3: 0x4e65,
		Data4: [8]byte{0xb9, 0x70, 0xc2, 0x56, 0x0f, 0xb5, 0xc2, 0x89},
	}
	// Microsoft-Windows-Kernel-Process provider GUID: "{22fb2cd6-0e7b-422b-a0c7-2fad1fd0e716}"
	// This provider generates process lifecycle events
	kernelProcessProviderGUID = windows.GUID{
		Data1: 0x22fb2cd6,
		Data2: 0x0e7b,
		Data3: 0x422b,
		Data4: [8]byte{0xa0, 0xc7, 0x2f, 0xad, 0x1f, 0xd0, 0xe7, 0x16},
	}
)

type fileObject uint64

type processStartKey uint64

// userInfo holds cached user and group information resolved from SIDs
type userInfo struct {
	sid         string
	domain      string
	name        string
	groupSID    string
	groupDomain string
	groupName   string
}

type etwReader struct {
	config  Config
	paths   map[string]struct{}
	parsers []FileParser

	log        *logp.Logger
	etwSession *etw.Session

	done       <-chan struct{}
	eventC     chan Event
	stopC      chan struct{}
	inflightWG sync.WaitGroup

	deviceResolver *deviceResolver

	pathsCache    *lru.Cache[fileObject, string]
	processCache  *lru.Cache[processStartKey, *Process]
	sidCache      *lru.Cache[string, *userInfo]
	fileInfoCache *lru.Cache[string, os.FileInfo]

	correlator *operationsCorrelator

	// PID of the ETW reader process, used for filtering events
	ownpid uint32
	// Channel for ETW events, if needed for testing
	etwEventsC chan *etw.RenderedEtwEvent
}

func newETWReader(c Config, l *logp.Logger) (EventProducer, error) {
	l.Warn(cfgwarn.Preview("ETW file integrity monitoring may change in future releases."))

	paths := make(map[string]struct{})
	for _, p := range c.Paths {
		paths[p] = struct{}{}
	}

	deviceResolver, err := newDeviceResolver(l)
	if err != nil {
		return nil, fmt.Errorf("failed to create device resolver: %w", err)
	}

	pathsCache, err := lru.New[fileObject, string](1000)
	if err != nil {
		return nil, fmt.Errorf("failed to create paths cache: %w", err)
	}
	processCache, err := lru.New[processStartKey, *Process](1000)
	if err != nil {
		return nil, fmt.Errorf("failed to create process cache: %w", err)
	}
	sidCache, err := lru.New[string, *userInfo](1000)
	if err != nil {
		return nil, fmt.Errorf("failed to create SID cache: %w", err)
	}
	fileInfoCache, err := lru.New[string, os.FileInfo](1000)
	if err != nil {
		return nil, fmt.Errorf("failed to create file info cache: %w", err)
	}

	r := &etwReader{
		config:         c,
		parsers:        FileParsers(c),
		paths:          paths,
		eventC:         make(chan Event),
		deviceResolver: deviceResolver,
		pathsCache:     pathsCache,
		processCache:   processCache,
		sidCache:       sidCache,
		fileInfoCache:  fileInfoCache,
		correlator:     newOperationsCorrelator(),
		ownpid:         windows.GetCurrentProcessId(), // Get the PID of the current process
	}

	r.etwSession, err = etw.NewSession(etw.Config{
		SessionName: "AuditbeatFIMSession",
		Providers: []etw.ProviderConfig{
			{
				Name: "Microsoft-Windows-Kernel-File",
				EnableProperty: []string{
					"EVENT_ENABLE_PROPERTY_SID",
					"EVENT_ENABLE_PROPERTY_PROCESS_START_KEY",
				},
				MatchAnyKeyword: fileKeywords,
				EventFilter: etw.EventFilter{
					EventIDs: []uint16{
						fileNameCreate, fileCreate,
						fileClose, fileWrite,
						fileSetInformation, fileDeletePath,
						fileRenamePath, fileSetLinkPath,
						fileSetSecurity, fileSetEA,
					},
					FilterIn: true,
				},
			},
			{
				Name: "Microsoft-Windows-Kernel-Process",
				EnableProperty: []string{
					"EVENT_ENABLE_PROPERTY_SID",
					"EVENT_ENABLE_PROPERTY_PROCESS_START_KEY",
				},
				MatchAnyKeyword: WINEVENT_KEYWORD_PROCESS,
				EventFilter: etw.EventFilter{
					EventIDs: []uint16{processStart, processStop},
					FilterIn: true,
				},
			},
		},
		BufferSize:     1024,
		MinimumBuffers: 8,
		MaximumBuffers: 16,
	})
	if err != nil {
		return nil, fmt.Errorf("error initializing ETW session: %w", err)
	}
	r.etwSession.Callback = r.consumeEvent

	r.log = l.With("etw_session", r.etwSession.Name)

	// Create a new realtime session
	// If it fails with ERROR_ALREADY_EXISTS we try to attach to it
	createErr := r.etwSession.CreateRealtimeSession()
	if createErr != nil {
		if !errors.Is(createErr, etw.ERROR_ALREADY_EXISTS) {
			return nil, fmt.Errorf("realtime session could not be created: %w", createErr)
		}
		r.log.Debug("session already exists, trying to attach to it")
		// Attach to an existing session
		if err := r.etwSession.AttachToExistingSession(); err != nil {
			return nil, fmt.Errorf("unable to retrieve handler: %w", err)
		}
		r.log.Debug("attached to existing session")
	} else {
		r.log.Debug("created new session")
	}
	return r, nil
}

func (r *etwReader) Start(done <-chan struct{}) (<-chan Event, error) {
	r.done = done
	r.stopC = make(chan struct{})
	go func() {
		if err := r.etwSession.StartConsumer(); err != nil {
			r.log.Errorf("failed running ETW consumer: %v", err)
		}
	}()

	var flushWg sync.WaitGroup
	flushWg.Add(1)
	go func() {
		interval := r.config.FlushInterval
		timer := time.NewTicker(interval)
		defer timer.Stop()
		defer flushWg.Done()
		for {
			select {
			case <-r.done:
				return
			case <-timer.C:
				ops := r.correlator.flushExpiredGroups(interval)
				for _, op := range ops {
					r.sendEvent(op)
				}
			}
		}
	}()

	go func() {
		defer r.pathsCache.Purge()
		defer r.processCache.Purge()
		defer r.sidCache.Purge()
		defer r.fileInfoCache.Purge()

		<-r.done

		// Stop the ETW session, which flushes buffers and causes StartConsumer to exit.
		if err := r.etwSession.StopSession(); err != nil {
			r.log.Errorf("failed to stop ETW session: %v", err)
		} else {
			r.log.Debug("ETW session stopped")
		}

		close(r.stopC)
		// Wait for the main consumer loop and all in-flight events to finish processing.
		r.inflightWG.Wait()
		flushWg.Wait()

		// flush any remaining operations
		ops := r.correlator.flushExpiredGroups(0)
		for _, op := range ops {
			r.sendEvent(op)
		}

		close(r.eventC)
		r.eventC = nil
	}()

	r.log.Infow("started etw watcher", "file_path", r.config.Paths, "recursive", r.config.Recursive)

	return r.eventC, nil
}

func (r *etwReader) consumeEvent(record *etw.EventRecord) uintptr {
	select {
	case <-r.stopC:
		return 0
	default:
	}

	// we ignore events from our own process to avoid self-generated events
	if record.EventHeader.ProcessId == r.ownpid {
		r.log.Debugf("ignoring event from our own process: %d", r.ownpid)
		return 0
	}

	// Add to WaitGroup to signal that an event is being processed.
	r.inflightWG.Add(1)
	defer r.inflightWG.Done()

	switch record.EventHeader.ProviderId {
	case kernelFileProviderGUID:
		return r.handleFileEvent(record)
	case kernelProcessProviderGUID:
		return r.handleProcessEvent(record)
	}
	return 0
}

func (r *etwReader) handleFileEvent(record *etw.EventRecord) uintptr {
	if skipFileEvent(record.EventHeader.EventDescriptor.Id) {
		// should never happen, but there are some systems that might not support
		// the event filtering properly and we want to avoid unnecessary processing
		return 0
	}

	etwEvent, err := r.etwSession.RenderEvent(record)
	if err != nil {
		if errors.Is(err, etw.ErrUnprocessableEvent) {
			return 0
		}
		r.log.Errorw("failed to render ETW event", "error", err)
		return 1
	}

	switch etwEvent.EventID {
	case fileCreate, fileRenamePath, fileSetLinkPath:
		if !r.cacheFilename(&etwEvent) {
			return 0
		}
	}

	path := r.getEventPath(&etwEvent)
	if path == "" {
		return 0
	}

	if r.excluded(path) {
		return 0
	}

	for _, op := range r.correlator.processEvent(path, &etwEvent) {
		r.sendEvent(op)
	}

	if etwEvent.EventID == fileClose {
		_ = r.evictFilename(&etwEvent)
	}

	if r.etwEventsC != nil {
		r.etwEventsC <- &etwEvent
	}
	return 0
}

func (r *etwReader) cacheFilename(etwEvent *etw.RenderedEtwEvent) bool {
	fileObj := fileObject(getUint64Property(etwEvent, "FileObject"))
	path := r.deviceResolver.translateDevicePath(getRawPathFromEvent(etwEvent))
	if r.excluded(path) {
		return false
	}
	if fileObj != 0 {
		_ = r.pathsCache.Add(fileObj, path)
	}
	return true
}

func (r *etwReader) evictFilename(etwEvent *etw.RenderedEtwEvent) bool {
	fileObj := fileObject(getUint64Property(etwEvent, "FileObject"))
	if fileObj != 0 && r.pathsCache.Contains(fileObj) {
		return r.pathsCache.Remove(fileObj)
	}
	return false
}

func (r *etwReader) getEventPath(etwEvent *etw.RenderedEtwEvent) string {
	var path string
	fileObj := fileObject(getUint64Property(etwEvent, "FileObject"))
	if fileObj != 0 {
		var found bool
		path, found = r.pathsCache.Get(fileObj)
		if found {
			return path
		}
	}
	return r.deviceResolver.translateDevicePath(getRawPathFromEvent(etwEvent))
}

func (r *etwReader) sendEvent(op *etwOp) {
	if op == nil || op.action == None {
		return
	}

	start := time.Now()

	path := r.deviceResolver.translateDevicePath(op.path)
	info, updated, infoErr := r.getFileInfo(path)

	switch op.action {
	case AttributesModified:
		if info == nil || !updated {
			// ignore AttributesModified events if we couldn't
			// get file info or if it hasn't changed to avoid noise
			return
		}
	case Deleted:
		r.clearFileInfoCache(path)
	}

	event := NewEventFromFileInfo(
		path,
		info,
		infoErr,
		op.action,
		SourceETW,
		r.config.MaxFileSizeBytes,
		r.config.HashTypes,
		r.parsers,
	)

	event.Timestamp = op.end
	event.Process = r.getProcess(op)
	if event.Info != nil {
		event.Info.ExtendedAttributes = readExtendedAttributes(path)
	}
	event.rtt = time.Since(start)

	if r.eventC != nil {
		r.eventC <- event
	}
}

func (r *etwReader) getFileInfo(path string) (info os.FileInfo, updated bool, err error) {
	info, err = os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// deleted/moved file is signaled by info == nil
			return nil, false, nil
		} else {
			return nil, false, fmt.Errorf("failed to lstat: %w", err)
		}
	}

	// Check if we have cached file info for this path
	cachedInfo, found := r.fileInfoCache.Get(path)
	if !found {
		r.fileInfoCache.Add(path, info)
		return info, true, nil
	}

	// Compare current info with cached info
	updated = r.fileInfoChanged(cachedInfo, info)
	if updated {
		r.fileInfoCache.Add(path, info)
	}

	return info, updated, nil
}

// fileInfoChanged compares two os.FileInfo objects to determine if file attributes have changed
func (r *etwReader) fileInfoChanged(cached, current os.FileInfo) bool {
	if cached.Size() != current.Size() {
		return true
	}

	// Modification time changed (with small tolerance for filesystem precision)
	if !cached.ModTime().Truncate(time.Millisecond).Equal(current.ModTime().Truncate(time.Millisecond)) {
		return true
	}

	if cached.Mode() != current.Mode() {
		return true
	}

	if cached.IsDir() != current.IsDir() {
		return true
	}

	cachedSys, okCached := cached.Sys().(*windows.Win32FileAttributeData)
	currentSys, okCurrent := current.Sys().(*windows.Win32FileAttributeData)
	if cachedSys != nil && currentSys != nil && okCached && okCurrent {
		if *cachedSys != *currentSys {
			return true
		}
	}

	return false
}

func getRawPathFromEvent(etwEvent *etw.RenderedEtwEvent) string {
	path := getStringProperty(etwEvent, "FileName")
	if path == "" {
		path = getStringProperty(etwEvent, "FilePath")
	}
	if path == "" {
		return ""
	}
	return path
}

func (r *etwReader) excluded(path string) bool {
	if r.config.IsExcludedPath(path) {
		return true
	}

	if !r.config.IsIncludedPath(path) {
		return true
	}

	dir := filepath.Dir(path)

	if !r.config.Recursive {
		if _, ok := r.paths[dir]; ok {
			return false
		}
	} else {
		for p := range r.paths {
			if strings.HasPrefix(dir, p) {
				return false
			}
		}
	}

	return true
}

func (r *etwReader) handleProcessEvent(record *etw.EventRecord) uintptr {
	if skipProcessEvent(record.EventHeader.EventDescriptor.Id) {
		// should never happen, but there are some systems that might not support
		// the event filtering properly and we want to avoid unnecessary processing
		return 0
	}

	etwEvent, err := r.etwSession.RenderEvent(record)
	if err != nil {
		if errors.Is(err, etw.ErrUnprocessableEvent) {
			return 0
		}
		r.log.Errorf("failed to render ETW event: %v", err)
		return 1
	}

	startKey := processStartKey(getUint64ExtendedData(&etwEvent, "PROCESS_START_KEY"))

	switch etwEvent.EventID {
	case processStart:
		var process Process
		process.Name = r.deviceResolver.translateDevicePath(getStringProperty(&etwEvent, "ImageName"))
		process.User.ID = getStringExtendedData(&etwEvent, "SID")
		userInfo := r.getUserInfo(process.User.ID)
		if userInfo != nil {
			process.User.Domain = userInfo.domain
			process.User.Name = userInfo.name
			process.Group.ID = userInfo.groupSID
			process.Group.Domain = userInfo.groupDomain
			process.Group.Name = userInfo.groupName
		}
		process.PID = getUint32Property(&etwEvent, "ProcessID")
		createTime := getDateTimeProperty(&etwEvent, "CreateTime", time.RFC3339Nano)
		process.EntityID = fmt.Sprintf("%d-%d", process.PID, createTime.UnixNano())
		_ = r.processCache.Add(startKey, &process)
	case processStop:
		_ = r.processCache.Remove(startKey)
	}
	return 0
}

func (r *etwReader) processFromOp(op *etwOp) *Process {
	process := &Process{
		PID: op.pid,
	}
	process.User.ID = op.sid
	userInfo := r.getUserInfo(process.User.ID)
	if userInfo != nil {
		process.User.Domain = userInfo.domain
		process.User.Name = userInfo.name
		process.Group.ID = userInfo.groupSID
		process.Group.Domain = userInfo.groupDomain
		process.Group.Name = userInfo.groupName
	}
	return process
}

func (r *etwReader) getProcess(op *etwOp) *Process {
	process, found := r.processCache.Get(op.processStartKey)
	if !found {
		// Fallback to using the basic process information
		// available in the ETW event.
		return r.processFromOp(op)
	}
	return process
}

func (r *etwReader) getUserInfo(sidStr string) *userInfo {
	if sidStr == "" {
		return nil
	}

	if cachedInfo, found := r.sidCache.Get(sidStr); found {
		return cachedInfo
	}

	sid, err := windows.StringToSid(sidStr)
	if err != nil {
		r.log.Errorw("failed to convert string to SID", "sid_string", sidStr, "error", err)
		// we cache failed ones also to avoid repeated lookups
		_ = r.sidCache.Add(sidStr, nil)
		return nil
	}
	var info userInfo
	info.sid = sidStr
	var use uint32
	info.name, info.domain, use, err = sid.LookupAccount("")
	if err != nil {
		// we do not cache this one since it might be a temporary error
		r.log.Errorw("failed to lookup account for SID", "sid", sidStr, "error", err)
		return &info
	}

	switch use {
	case windows.SidTypeGroup, windows.SidTypeWellKnownGroup, windows.SidTypeAlias:
		info.groupSID = info.sid       // If it's a group, use the same SID
		info.groupDomain = info.domain // If it's a group, use the same domain
		info.groupName = info.name     // If it's a group, use the same name
	default:
		// Build the full account name for NetUserGetInfo API
		fullAccountName := info.name
		if info.domain != "" {
			fullAccountName = fmt.Sprintf("%s\\%s", info.domain, info.name)
		}
		info.groupSID, info.groupDomain, info.groupName = getPrimaryGroupForAccount(sidStr, fullAccountName)
	}
	_ = r.sidCache.Add(sidStr, &info)
	return &info
}

// userInfo4 is a Go representation of the C struct USER_INFO_4.
type userInfo4 struct {
	Name            *uint16
	Password        *byte
	PasswordAge     uint32
	Priv            uint32
	HomeDir         *uint16
	Comment         *uint16
	Flags           uint32
	ScriptPath      *uint16
	AuthFlags       uint32
	FullName        *uint16
	UserComment     *uint16
	Params          *uint16
	Workstations    *uint16
	LastLogon       uint32
	LastLogoff      uint32
	AcctExpires     uint32
	MaxStorage      uint32
	UnitsPerWeek    uint32
	LogonHours      *byte
	BadPwCount      uint32
	NumLogons       uint32
	LogonServer     *uint16
	CountryCode     uint32
	CodePage        uint32
	UserSid         *windows.SID
	PrimaryGroupId  uint32
	Profile         *uint16
	HomeDirDrive    *uint16
	PasswordExpired uint32
}

func getPrimaryGroupForAccount(sid, account string) (groupSID, groupDomain, groupName string) {
	accountPtr, err := windows.UTF16PtrFromString(account)
	if err != nil {
		return "", "", ""
	}

	var buf *byte
	err = windows.NetUserGetInfo(nil, accountPtr, 4, &buf)
	if err != nil {
		return "", "", ""
	}
	defer windows.NetApiBufferFree(buf)

	userInfo := (*userInfo4)(unsafe.Pointer(buf))
	primaryGroupRID := userInfo.PrimaryGroupId

	lastHyphen := strings.LastIndex(sid, "-")
	if lastHyphen == -1 {
		return "", "", ""
	}
	domainSIDString := sid[:lastHyphen]
	groupSID = fmt.Sprintf("%s-%d", domainSIDString, primaryGroupRID)

	primaryGroupSID, err := windows.StringToSid(groupSID)
	if err != nil {
		// If lookup fails, we can still return the SID string.
		return groupSID, "", ""
	}

	groupName, domain, _, err := primaryGroupSID.LookupAccount("")
	if err != nil {
		return groupSID, "", ""
	}

	if groupName == "None" {
		return "", "", ""
	}

	return groupSID, domain, groupName
}

func skipFileEvent(event uint16) bool {
	switch event {
	case fileNameCreate,
		fileCreate,
		fileClose,
		fileWrite,
		fileSetInformation,
		fileDeletePath,
		fileRenamePath,
		fileSetLinkPath,
		fileSetSecurity,
		fileSetEA:
		return false
	default:
		return true
	}
}

func skipProcessEvent(event uint16) bool {
	switch event {
	case processStart, processStop:
		return false
	default:
		return true
	}
}

func getUint64Property(ee *etw.RenderedEtwEvent, name string) uint64 {
	value, _ := ee.GetProperty(name).(string)
	if value == "" {
		return 0
	}
	return parseUint64(value)
}

func parseUint64(value string) uint64 {
	base := 10
	if strings.HasPrefix(value, "0x") {
		base = 16
		value = strings.TrimPrefix(value, "0x")
	}
	num, _ := strconv.ParseUint(value, base, 64)
	return num
}

func getUint64ExtendedData(ee *etw.RenderedEtwEvent, name string) uint64 {
	value, _ := ee.GetExtendedData(name).(string)
	if value == "" {
		return 0
	}
	return parseUint64(value)
}

func getUint32Property(ee *etw.RenderedEtwEvent, name string) uint32 {
	return uint32(getUint64Property(ee, name))
}

func getStringProperty(ee *etw.RenderedEtwEvent, name string) string {
	value, _ := ee.GetProperty(name).(string)
	return value
}

func getDateTimeProperty(ee *etw.RenderedEtwEvent, name, format string) time.Time {
	value, _ := ee.GetProperty(name).(string)
	if value == "" {
		return time.Time{}
	}
	t, err := time.Parse(format, value)
	if err != nil {
		return time.Time{}
	}
	return t
}

func getStringExtendedData(ee *etw.RenderedEtwEvent, name string) string {
	value, _ := ee.GetExtendedData(name).(string)
	return value
}

// clearFileInfoCache removes file info from cache when file is deleted
func (r *etwReader) clearFileInfoCache(path string) {
	if r.fileInfoCache.Contains(path) {
		r.fileInfoCache.Remove(path)
		r.log.Debugw("Cleared file info cache for deleted file", "path", path)
	}
}
