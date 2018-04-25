package main

import (
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
	"strings"
)

type (
	Watcher struct {
		*fsnotify.Watcher
		events chan fsnotify.Event
		Logger
	}
)

func (w *Watcher) WalkFunc(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		if strings.HasPrefix(filepath.Base(info.Name()), ".") {
			return filepath.SkipDir
		}
		w.Add(path)
	}
	return nil
}

func NewWatcher(logger Logger) (*Watcher, error) {
	watcher := &Watcher{nil, make(chan fsnotify.Event), logger}

	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	if watcher.Watcher, err = fsnotify.NewWatcher(); err != nil {
		return nil, err
	}

	if err := filepath.Walk(dir, watcher.WalkFunc); err != nil {
		watcher.Close()
		return nil, err
	}
	go watcher.start()
	return watcher, nil
}

func (w *Watcher) start() {
	for event := range w.Watcher.Events {
		if event.Op == fsnotify.Remove || event.Op == fsnotify.Rename {
			event.Op = fsnotify.Remove
			w.events <- event
		}
		info, err := os.Stat(event.Name)
		if err != nil {
			w.Log(ERROR, "Failed to stat event target file: %q", err)
		}
		if !info.IsDir() {
			if event.Op == fsnotify.Write {
				w.events <- event
			}
		} else if event.Op == fsnotify.Create {
			w.events <- event
		}
	}
}

func (w *Watcher) Start(cb func(path string)) { // start watching events
	for {
		select {
		case event := <-w.events:
			switch event.Op {
			case fsnotify.Write:
				cb(event.Name)
			case fsnotify.Create:
				if err := filepath.Walk(event.Name, w.WalkFunc); err != nil {
					w.Log(ERROR, "Failed to walk newly created directory: %q", err)
				}
			case fsnotify.Remove:
				w.Remove(event.Name)
			}
		case err := <-w.Errors:
			w.Log(ERROR, "Caught notify error: %q", err)
		}
	}
}
