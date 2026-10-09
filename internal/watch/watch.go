package watch

import (
	"github.com/fsnotify/fsnotify"
	"sync"
	"time"
)

type Coordinator struct {
	watcher *fsnotify.Watcher
	mu      sync.Mutex
	paths   map[string]bool
	done    chan struct{}
	once    sync.Once
	failed  bool
	change  func()
}

func New(change func()) (*Coordinator, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	c := &Coordinator{watcher: watcher, paths: map[string]bool{}, done: make(chan struct{}), change: change}
	go c.run()
	return c, nil
}
func (c *Coordinator) Set(paths []string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	next := map[string]bool{}
	c.failed = false
	for _, path := range paths {
		next[path] = true
	}
	for path := range c.paths {
		if !next[path] {
			_ = c.watcher.Remove(path)
			delete(c.paths, path)
		}
	}
	for path := range next {
		if c.paths[path] {
			continue
		}
		if len(c.paths) >= 8000 {
			c.failed = true
			continue
		}
		if err := c.watcher.Add(path); err != nil {
			c.failed = true
			continue
		}
		c.paths[path] = true
	}
	return !c.failed
}
func (c *Coordinator) run() {
	timer := time.NewTimer(time.Hour)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()
	reconcile := time.NewTicker(15 * time.Second)
	defer reconcile.Stop()
	last := time.Now()
	ticks := 0
	for {
		select {
		case event, ok := <-c.watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
				c.mu.Lock()
				delete(c.paths, event.Name)
				c.mu.Unlock()
			}
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename|fsnotify.Chmod) != 0 {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(200 * time.Millisecond)
			}
		case _, ok := <-c.watcher.Errors:
			if !ok {
				return
			}
			c.mu.Lock()
			c.failed = true
			c.mu.Unlock()
			c.change()
		case <-timer.C:
			c.change()
		case now := <-reconcile.C:
			c.mu.Lock()
			failed := c.failed
			c.mu.Unlock()
			ticks++
			if now.Sub(last) > 30*time.Second || (failed && ticks%4 == 0) {
				c.change()
			}
			last = now
		case <-c.done:
			return
		}
	}
}
func (c *Coordinator) Close() { c.once.Do(func() { close(c.done); _ = c.watcher.Close() }) }
