package globals

import (
	"context"
	"sync"

	"k8s.io/client-go/dynamic"

	notifikv1alpha1 "freepik.com/notifik/api/v1alpha1"
)

var (
	Application = applicationT{
		Context: context.Background(),

		WatcherPool: make(map[ResourceTypeName]ResourceTypeWatcherT),
	}
)

// TODO
type ResourceTypeName string

// ApplicationT TODO
type applicationT struct {
	// Context TODO
	Context context.Context

	// Configuration TODO
	Configuration notifikv1alpha1.ConfigurationT

	// KubeRawClient TODO
	KubeRawClient *dynamic.DynamicClient

	// WatcherPool TODO
	WatcherPool map[ResourceTypeName]ResourceTypeWatcherT
}

// TODO
type ResourceTypeWatcherT struct {
	// Enforce concurrency safety
	Mutex *sync.Mutex

	//
	Started    *bool
	StopSignal *chan bool

	//
	NotificationList *[]*notifikv1alpha1.Notification
}
