package watcher

import (
	"gobackup/pkg/models"
	"time"
)

/*
Debouncer:
  - reset time if u see an event in 500 ms.
  - if u see an event in 500 ms, do not send
  - send after 500 ms of no events
*/
func (w *Watcher) debouncedSend(path string, fn func()) {
	w.debounceMu.Lock()
	defer w.debounceMu.Unlock()

	if timer, exists := w.debouncer[path]; exists {
		timer.Stop()
	}

	w.debouncer[path] = time.AfterFunc(500*time.Millisecond, func() {
		fn()
		w.debounceMu.Lock()
		delete(w.debouncer, path)
		w.debounceMu.Unlock()
	})

}

func (w *Watcher) Changes() <-chan models.FileEvent {
	return w.changeChan
}

func (w *Watcher) Errors() <-chan error {
	return w.errorChan
}

func (w *Watcher) Close() error {
	w.cancel()
	return w.fsNotifyWatcher.Close()
}
