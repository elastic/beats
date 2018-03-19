# Really quick FSEvents overview

For those new to FSEvents itself (the Apple technology), here's a really quick primer:

Introduced in macOS v10.5, File System Events is made up of three parts:

* kernel code that passes raw event data to user space through a special device
* a daemon that filters this stream and sends notifications
* a database that stores a record of all changes

The API provides access to this system allowing applications to be notified of changes to directory hierarchies, and also to retrieve a history of changes. File System Events are path / file name based, *not* file descriptor based. This means that if you watch a directory (path) and it is moved elsewhere your watch does not automatically move with it.

The API uses the concept of a *Stream* (FSEventStream) which is a configurable stream of notifications. A Stream can recursively monitor many directories. Options like latency which allow the coalescing of events are specific to one stream.

## Persistent Monitoring

The database kept by the File System Events API allow a program that is run periodically to check for changes to the file system between runs. This has a real impact on the API and why there are parameters like `since`. This feature should be considered *advisory* according to Apple's docs - a full scan should still be run periodically. If an older version of macOS modifies the file system (say, by removing the drive and putting it in another computer) it would not update the database.

The `since` parameter must be an EventID for the host or device that the stream is for (or the special ALL or NOW values).

To discover the EventID at a specific time the function `LastEventBefore` can be used (calls `FSEventsGetLastEventIdForDeviceBeforeTime`); provide dev==0 for a host EventID. Current() (calls FSEventsGetCurrentEventId) returns the most recent EventID.

## Apple Docs

* [FSEvents_ProgGuide.pdf][https://developer.apple.com/library/mac/documentation/Darwin/Conceptual/FSEvents_ProgGuide/FSEvents_ProgGuide.pdf]
* [FSEvents_ProgGuide html][https://developer.apple.com/library/mac/documentation/Darwin/Conceptual/FSEvents_ProgGuide/Introduction/Introduction.html#//apple_ref/doc/uid/TP40005289-CH1-SW1]

## Device Streams vs Host Streams

A Host Stream (dev == 0) can monitor directories through the entire system (assuming appropriate permissions). EventIDs increase across all devices (i.e. an older event on any device has a smaller EventID than a newer one except when EventID's rollover). This means that if another disk is added / mounted then there is the potential for a historical EventID conflict.

A Device Stream (pass a `Device` to `New`) can monitor directories only on the given device. Because the EventIDs refer only to that device there is no chance of conflict.

For real-time monitoring there aren't any notable advantages to a Device Stream over a Host stream. For persistent monitoring (run a program today to see what changed yesterday) Device Streams are more robust since there can be no EventID conflict.

## File Events

Is macOS v10.7 Apple introduced *File Events* (kFSEventStreamCreateFlagFileEvents aka CF_FILEEVENTS). Prior to this events are delivered for directories only, i.e. if /tmp/subdir/testfile is modified and an FSEventStream is monitoring /tmp then an event would be delivered for the path /tmp/subdir. This tells the application that it should scan /tmp/subdir for changes. It was also possible (in corner cases) that an event for the path /tmp could be created with MUSTSCANSUBDIRS set. This should only happen if events are being dropped, and had to be coalesced to prevent the loss of information.

With CF_FILEEVENTS set (>= macOS v10.7) events are generated with the path specifying the individual files that have been modified. I haven't found explicit mention, but it seems likely that the same caveats apply with respect to coalescing if events are being dropped.

Apple warns that using File Events will cause many more events to be generated (in part, I expect, because they don't coalesce as easily as directory-level events).

## Temporal Coalescing

If two files in the same directory are changed in a short period of time (e.g. /tmp/test1 and /tmp/test2) (assuming CF_FILEEVENTS=0) a single event could be delivered to the application specifying that the path `/tmp` contains changes. There is an efficiency boost when scanning /tmp only once looking for all changes versus scanning it once for test1 and once for test2. The `latency` parameter enables this kind of temporal coalescing. If `latency` is set to, say, 1 second then the application will not be notified more than once a second about changes. The flag kFSEventStreamCreateFlagNoDefer aka CF_NODEFER specifies whether the application is notified on the leading or lagging edge of changes.

# Structure of fsevents.go

### Creating a Stream

`EventStream` encapsulates an FSEventStream, and allows an arbitrary number of paths to be monitored.

For real-time monitoring an EventStream is created with `Resume` == `false`. This means it will not deliver historical events. If `Resume` == `true` then all recorded events for the supplied paths since `EventId` would be supplied first, then realtime events would be supplied as they occur.

The `Latency` parameter is passed on to the API, and used to throttle / coalesce events - '0' means deliver all events.

`Device` can be used to create streams specific to a device, (See Device Streams vs Host Streams). Use '0' to create a host stream.

Instantiating an `EventStream` just allocates memory to store the *configuration* of the stream. Calling `EventStream.Start` creates the FSEventStream, and a channel that will be used to report events `EventStream.Events` (unless one has already been supplied). The EventStream is stored (via an unsafe.Pointer) in the FSEventStream context (so the OS callback has access to it).

### Running a Stream

In macOS an application must run a RunLoop (see: [Run Loop Management][https://developer.apple.com/library/mac/documentation/cocoa/Conceptual/Multithreading/RunLoopManagement/RunLoopManagement.html#//apple_ref/doc/uid/10000057i-CH16-SW1]) for each thread that wishes to receive events from the operating system.

An FSEventStream needs to be started, and must be supplied with a Runloop reference to deliver events to.

To accomplish this `EventStream.Start` creates a goroutine that calls CFRunLoopRun(). The call in to CFRunLoopRun does not return until the runloop is stopped by `EventStream.Stop`. I _believe_ that the current implementation will work correctly because of the following:

* There is a many->one mapping from goroutines to os threads (i.e. goroutines do not switch os-threads).
* The os thread that runs the Run Loop will not be available for other goroutines because CFRunLoopRun does not return.
* This therefore means that each call to `EventStream.Start` will consume an OS thread until the runloop is stopped by `EventStream.Stop` calling CFRunLoopStop. Each Stream will get it's own Run Loop (on it's own OS thread).

### Receiving Events

The File System Events API causes a C-callback to be called by delivering an event. The callback extracts / converts the supplied data then posts an array of `Event`s on the channel `EventStream.Events`. The `EventStream` is stored in the context parameter of the FSEventStream (supplied back to the callback by the File System Events API).

### Stopping a Stream

`EventStream.Stop` stops and invalidates the stream (as per the File System Events API requirements), then releases the Stream memory, and stops the runloop that was started by `EventStream.Start`. This returns the `EventStream` to (almost) the same state as before `EventStream.Start` was called. The difference is that `EventStream.EventID` is now set to the last event ID received before the stream was stopped, and `EventStream.Events` will now reference an instantiated channel (it may have been nil before the call to `EventStream.Start`).

### Memory / Callback Safety

`EventStream` manages some OS resources (an OS thread, an FSEventStream, an OS runloop, etc). If the `EventStream` is garbage collected without the being stopped then a panic is likely. This is caused by an FSEventStream callback attempting to reference the `EventStream` via an `unsafe.Pointer`. To guard against this situation a finalizer is attached to the `EventStream` when it is started. When the finalizer triggers (through GC of the `EventStream`) it stops the stream so that the OS resources can be freed. This prevents the panic.
