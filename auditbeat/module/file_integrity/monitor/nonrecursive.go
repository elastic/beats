package monitor

import "github.com/fsnotify/fsnotify"

type nonRecursiveWatcher fsnotify.Watcher

func (*nonRecursiveWatcher) Start() error {
	return nil
}

func (watcher *nonRecursiveWatcher) Add(path string) error {
	return (*fsnotify.Watcher)(watcher).Add(path)
}

func (watcher *nonRecursiveWatcher) Close() error {
	return (*fsnotify.Watcher)(watcher).Close()
}

func (watcher *nonRecursiveWatcher) EventChannel() <-chan fsnotify.Event {
	return (*fsnotify.Watcher)(watcher).Events
}

func (watcher *nonRecursiveWatcher) ErrorChannel() <-chan error {
	return (*fsnotify.Watcher)(watcher).Errors
}
