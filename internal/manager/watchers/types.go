package watchers

import "sync"

type ResourceTypeName = string

// ResourceWatcher wraps status and control of a watcher of a resource.
type ResourceWatcher struct {
	mu sync.Mutex

	Started    bool
	StopSignal chan bool
}

// WatchersManager manage watchers' lifecycle
type WatchersManager struct {
	mu       sync.Mutex
	watchers map[ResourceTypeName]*ResourceWatcher
}
