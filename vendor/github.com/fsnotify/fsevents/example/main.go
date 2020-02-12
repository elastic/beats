// +build darwin

package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/fsnotify/fsevents"
)

func main() {
	path, err := ioutil.TempDir("", "fsexample")
	if err != nil {
		log.Fatalf("Failed to create TempDir: %v", err)
	}
	dev, err := fsevents.DeviceForPath(path)
	if err != nil {
		log.Fatalf("Failed to retrieve device for path: %v", err)
	}
	log.Print(dev)
	log.Println(fsevents.EventIDForDeviceBeforeTime(dev, time.Now()))

	es := &fsevents.EventStream{
		Paths:   []string{path},
		Latency: 500 * time.Millisecond,
		Device:  dev,
		Flags:   fsevents.FileEvents | fsevents.WatchRoot}
	es.Start()
	ec := es.Events

	log.Println("Device UUID", fsevents.GetDeviceUUID(dev))

	go func() {
		for msg := range ec {
			for _, event := range msg {
				logEvent(event)
			}
		}
	}()

	in := bufio.NewReader(os.Stdin)

	if false {
		log.Print("Started, press enter to GC")
		in.ReadString('\n')
		runtime.GC()
		log.Print("GC'd, press enter to quit")
		in.ReadString('\n')
	} else {
		log.Print("Started, press enter to stop")
		in.ReadString('\n')
		es.Stop()

		log.Print("Stopped, press enter to restart")
		in.ReadString('\n')
		es.Resume = true
		es.Start()

		log.Print("Restarted, press enter to quit")
		in.ReadString('\n')
		es.Stop()
	}
}

var noteDescription = map[fsevents.EventFlags]string{
	fsevents.MustScanSubDirs: "MustScanSubdirs",
	fsevents.UserDropped:     "UserDropped",
	fsevents.KernelDropped:   "KernelDropped",
	fsevents.EventIDsWrapped: "EventIDsWrapped",
	fsevents.HistoryDone:     "HistoryDone",
	fsevents.RootChanged:     "RootChanged",
	fsevents.Mount:           "Mount",
	fsevents.Unmount:         "Unmount",

	fsevents.ItemCreated:       "Created",
	fsevents.ItemRemoved:       "Removed",
	fsevents.ItemInodeMetaMod:  "InodeMetaMod",
	fsevents.ItemRenamed:       "Renamed",
	fsevents.ItemModified:      "Modified",
	fsevents.ItemFinderInfoMod: "FinderInfoMod",
	fsevents.ItemChangeOwner:   "ChangeOwner",
	fsevents.ItemXattrMod:      "XAttrMod",
	fsevents.ItemIsFile:        "IsFile",
	fsevents.ItemIsDir:         "IsDir",
	fsevents.ItemIsSymlink:     "IsSymLink",
}

func logEvent(event fsevents.Event) {
	note := ""
	for bit, description := range noteDescription {
		if event.Flags&bit == bit {
			note += description + " "
		}
	}
	log.Printf("EventID: %d Path: %s Flags: %s", event.ID, event.Path, note)
}
