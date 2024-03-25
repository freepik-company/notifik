package globals

import (
	"context"
	jokativ1alpha1 "freepik.com/jokati/api/v1alpha1"
	"k8s.io/client-go/dynamic"
)

var (
	Application = applicationT{
		Context: context.Background(),
		//KubernetesRawClient: NewClient(),
		WatcherPool: make(map[ResourceTypeName]ResourceTypeWatcherT),
	}
)

// TODO
type ResourceTypeName string

// ApplicationT TODO
type applicationT struct {
	Context context.Context

	//
	KubeRawClient *dynamic.DynamicClient

	// TODO
	WatcherPool map[ResourceTypeName]ResourceTypeWatcherT
}

// TODO
type ResourceTypeWatcherT struct {
	Started          bool
	NotificationList []*jokativ1alpha1.Notification
}
