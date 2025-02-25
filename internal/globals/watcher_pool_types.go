package globals

import (
	"freepik.com/notifik/api/v1alpha1"
	"sync"
)

// TODO
type ResourceTypeName string

// TODO
type ResourceTypeWatcherT struct {
	// Enforce concurrency safety
	Mutex *sync.Mutex

	// Started represents a flag to know if the watcher is running
	Started *bool
	// Blocked represents a flag to prevent watcher from starting
	Blocked *bool
	// StopSignal represents a flag to kill the watcher.
	// Watcher will be potentially re-launched by xyz.WorkloadController
	StopSignal *chan bool

	//
	NotificationList *[]*v1alpha1.Notification
}

type WatcherPoolT struct {
	// Enforce concurrency safety
	Mutex *sync.Mutex

	Pool map[ResourceTypeName]ResourceTypeWatcherT
}
