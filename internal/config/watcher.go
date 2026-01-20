package config

import (
	"log"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches a config file for changes and reloads it
type Watcher struct {
	path     string
	watcher  *fsnotify.Watcher
	mu       sync.RWMutex
	config   *Config
	handlers []func(*Config)
	done     chan struct{}
}

// NewWatcher creates a new config file watcher
func NewWatcher(path string) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Load initial config
	cfg, err := Load(path)
	if err != nil {
		w.Close()
		return nil, err
	}

	cw := &Watcher{
		path:    path,
		watcher: w,
		config:  cfg,
		done:    make(chan struct{}),
	}

	// Add file to watcher
	if err := w.Add(path); err != nil {
		w.Close()
		return nil, err
	}

	return cw, nil
}

// Start starts watching for config file changes
func (w *Watcher) Start() {
	go w.watch()
}

// Stop stops the config watcher
func (w *Watcher) Stop() {
	close(w.done)
	w.watcher.Close()
}

// OnReload registers a handler to be called when config is reloaded
func (w *Watcher) OnReload(handler func(*Config)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers = append(w.handlers, handler)
}

// Get returns the current config
func (w *Watcher) Get() *Config {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.config
}

func (w *Watcher) watch() {
	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			// Reload on write or create (some editors do atomic saves via rename)
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				w.reload()
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Config watcher error: %v", err)
		}
	}
}

func (w *Watcher) reload() {
	cfg, err := Load(w.path)
	if err != nil {
		log.Printf("Failed to reload config: %v", err)
		return
	}

	w.mu.Lock()
	w.config = cfg
	handlers := make([]func(*Config), len(w.handlers))
	copy(handlers, w.handlers)
	w.mu.Unlock()

	log.Printf("Config reloaded from %s", w.path)

	// Notify handlers
	for _, handler := range handlers {
		handler(cfg)
	}
}
