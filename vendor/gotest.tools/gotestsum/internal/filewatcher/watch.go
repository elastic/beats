package filewatcher

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"gotest.tools/gotestsum/log"
)

const maxDepth = 7

func Watch(dirs []string, run func(pkg string) error) error {
	toWatch := findAllDirs(dirs, maxDepth)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close() // nolint: errcheck

	fmt.Printf("Watching %v directories. Use Ctrl-c to to stop a run or exit.\n", len(toWatch))
	for _, dir := range toWatch {
		if err = watcher.Add(dir); err != nil {
			return err
		}
	}

	timer := time.NewTimer(time.Hour)
	defer timer.Stop()

	h := &handler{last: time.Now(), fn: run}
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("exceeded idle timeout while watching files")
		case event := <-watcher.Events:
			if !timer.Stop() {
				<-timer.C
			}
			log.Debugf("handling event %v", event)

			if handleDirCreated(watcher, event) {
				continue
			}

			if err := h.handleEvent(event); err != nil {
				return fmt.Errorf("failed to run tests for %v: %v", event.Name, err)
			}
			timer.Reset(time.Hour)
		case err := <-watcher.Errors:
			return fmt.Errorf("failed while watching files: %v", err)
		}
	}
}

func findAllDirs(dirs []string, maxDepth int) []string {
	if len(dirs) == 0 {
		dirs = []string{"./..."}
	}

	var output []string // nolint: prealloc
	for _, dir := range dirs {
		const recur = "/..."
		if strings.HasSuffix(dir, recur) {
			dir = strings.TrimSuffix(dir, recur)
			output = append(output, findSubDirs(dir, maxDepth)...)
			continue
		}
		output = append(output, dir)
	}
	return output
}

func findSubDirs(rootDir string, maxDepth int) []string {
	var output []string
	// add root dir depth so that maxDepth is relative to the root dir
	maxDepth += pathDepth(rootDir)
	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warnf("failed to watch %v: %v", path, err)
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		if pathDepth(path) > maxDepth || exclude(path) {
			log.Debugf("Ignoring %v because of max depth or exclude list", path)
			return filepath.SkipDir
		}
		if !hasGoFiles(path) {
			log.Debugf("Ignoring %v because it has no .go files", path)
			return nil
		}
		output = append(output, path)
		return nil
	}
	// nolint: errcheck // error is handled by walker func
	filepath.Walk(rootDir, walker)
	return output
}

func pathDepth(path string) int {
	return strings.Count(filepath.Clean(path), string(filepath.Separator))
}

// return true if path is vendor, testdata, or starts with a dot
func exclude(path string) bool {
	base := filepath.Base(path)
	switch {
	case strings.HasPrefix(base, ".") && len(base) > 1:
		return true
	case base == "vendor" || base == "testdata":
		return true
	}
	return false
}

func hasGoFiles(path string) bool {
	fh, err := os.Open(path)
	if err != nil {
		return false
	}

	for {
		names, err := fh.Readdirnames(20)
		switch {
		case err == io.EOF:
			return false
		case err != nil:
			log.Warnf("failed to read directory %v: %v", path, err)
			return false
		}

		for _, name := range names {
			if strings.HasSuffix(name, ".go") {
				return true
			}
		}
	}
}

func handleDirCreated(watcher *fsnotify.Watcher, event fsnotify.Event) (handled bool) {
	if event.Op&fsnotify.Create != fsnotify.Create {
		return false
	}

	fileInfo, err := os.Stat(event.Name)
	if err != nil {
		log.Warnf("failed to stat %s: %s", event.Name, err)
		return false
	}

	if !fileInfo.IsDir() {
		return false
	}

	if err := watcher.Add(event.Name); err != nil {
		log.Warnf("failed to watch new directory %v: %v", event.Name, err)
	}
	return true
}

type handler struct {
	last time.Time
	fn   func(pkg string) error
}

const floodThreshold = 250 * time.Millisecond

func (h *handler) handleEvent(event fsnotify.Event) error {
	if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
		return nil
	}

	if !strings.HasSuffix(event.Name, ".go") {
		return nil
	}

	if time.Since(h.last) < floodThreshold {
		log.Debugf("skipping event received less than %v after the previous", floodThreshold)
		return nil
	}

	pkg := "./" + filepath.Dir(event.Name)
	fmt.Printf("\nRunning tests in %v\n", pkg)
	if err := h.fn(pkg); err != nil {
		return err
	}
	h.last = time.Now()
	return nil
}
