package globals

import (
	"context"
	"sync"

	//
	"k8s.io/client-go/dynamic"

	//
	"freepik.com/notifik/api/v1alpha1"
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

// ApplicationT TODO
type applicationT struct {
	// Context TODO
	Context context.Context

	// Configuration TODO
	Configuration v1alpha1.ConfigurationT

	// KubeRawClient TODO
	KubeRawClient *dynamic.DynamicClient

	// WatcherPool TODO
	//WatcherPool map[ResourceTypeName]ResourceTypeWatcherT
	WatcherPool WatcherPoolT
}
