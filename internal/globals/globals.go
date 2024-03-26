package globals

import (
	"context"

	"k8s.io/client-go/dynamic"

	jokativ1alpha1 "freepik.com/jokati/api/v1alpha1"
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
	Configuration jokativ1alpha1.ConfigurationT

	// KubeRawClient TODO
	KubeRawClient *dynamic.DynamicClient

	// WatcherPool TODO
	WatcherPool map[ResourceTypeName]ResourceTypeWatcherT
}

// TODO
type ResourceTypeWatcherT struct {
	Started          *bool
	NotificationList *[]*jokativ1alpha1.Notification
}
