package main

import (
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
	"strings"
)

type (
	op uint32
)

const (
	noop op = iota
	create
	write
	remove
)

func walkFunc(watcher *fsnotify.Watcher) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(filepath.Base(info.Name()), ".") {
				return filepath.SkipDir
			}

			watcher.Add(path)
		}
		return nil
	}
}

func newWatcher() (*fsnotify.Watcher, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err := filepath.Walk(dir, walkFunc(watcher)); err != nil {
		watcher.Close()
		return nil, err
	}
	return watcher, nil
}

func getOp(event fsnotify.Event) op {
	if event.Op == fsnotify.Remove || event.Op == fsnotify.Rename {
		return remove
	}
	info, err := os.Stat(event.Name)
	if err != nil {
		std.Printf("Failed to stat event target file: %q", err)
		return noop
	}
	if !info.IsDir() {
		if event.Op == fsnotify.Write {
			return write
		}
	} else if event.Op == fsnotify.Create {
		return create
	}
	return noop
}
