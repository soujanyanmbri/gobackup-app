package watcher

import (
	"context"
	"fmt"
	"gobackup/pkg/models"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// this will hanfle file watching via fsnotify

type Watcher struct {
	fsNotifyWatcher *fsnotify.Watcher
	watchedDirs     map[string]bool
	changeChan      chan models.FileEvent
	errorChan       chan error
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	debouncer       map[string]*time.Timer
	debounceMu      sync.Mutex
}

/*
Watcher:
1. File System Monitoring -
2. Recursive Directory Watching

Debouncer: 500ms debounce is added to prevent duplicate events. If multiple events occur for the same file - only the last event is processed, example: if i am writing this code, and i save it multiple times in quick succession, only the last save event is processed.
Concurrency is added, i added rwmutex to prevent concurrent access to the watchedDirs map - multiple reads OR single write.

Event Types:
CREATE, MODIFY, DELETE, SCAN events are generated.
*/

func NewWatcher() (*Watcher, error) {
	fsWacher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())

	return &Watcher{
		fsNotifyWatcher: fsWacher,
		watchedDirs:     make(map[string]bool),
		changeChan:      make(chan models.FileEvent),
		errorChan:       make(chan error, 10),
		ctx:             ctx,
		cancel:          cancel,
		mu:              sync.RWMutex{},
		debouncer:       make(map[string]*time.Timer),
	}, nil
}

func (w *Watcher) AddWatch(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Add directory to watch - recursive :)
			if err := w.fsNotifyWatcher.Add(walkPath); err != nil {
				return err
			}
			w.watchedDirs[walkPath] = true
			log.Printf("Watching directory: %s", walkPath)
		}
		return nil
	})
}

func (w *Watcher) Start() {
	go w.handleEvents()
	go w.periodicFullScan()
}
func (w *Watcher) handleEvents() {
	for {
		select {
		case <-w.ctx.Done():
			return
		case event, ok := <-w.fsNotifyWatcher.Events:
			if !ok {
				return
			}
			w.processEvent(event)
		case err, ok := <-w.fsNotifyWatcher.Errors:
			if !ok {
				return
			}
			w.errorChan <- err
		}
	}
}

func (w *Watcher) periodicFullScan() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.performFullScan()
		}
	}
}
func (w *Watcher) processEvent(event fsnotify.Event) {
	w.debouncedSend(event.Name, func() {
		var operation string
		switch {
		case event.Op&fsnotify.Create == fsnotify.Create:
			operation = "CREATE"
		case event.Op&fsnotify.Write == fsnotify.Write:
			operation = "MODIFY"
		case event.Op&fsnotify.Remove == fsnotify.Remove:
			operation = "DELETE"
		default:
			return
		}
		select {
		case w.changeChan <- models.FileEvent{
			Path:      event.Name,
			Operation: operation,
			Timestamp: time.Now(),
		}:
		case <-w.ctx.Done():
			return
		}
	})
}

func (w *Watcher) performFullScan() {
	w.mu.Lock()
	dirs := make([]string, 0, len(w.watchedDirs))
	for dir := range w.watchedDirs {
		dirs = append(dirs, dir)
	}
	w.mu.Unlock()

	for _, dir := range dirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {
				select {
				case w.changeChan <- models.FileEvent{
					Path:      path,
					Operation: "SCAN",
					Timestamp: time.Now(),
				}:
				case <-w.ctx.Done():
					return fmt.Errorf("context done")
				}
			}
			return nil
		})
	}

}
