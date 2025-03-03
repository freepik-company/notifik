package notifications

import (
	"freepik.com/notifik/api/v1alpha1"
	"sync"
)

// TODO
type ResourceTypeName = string

type NotificationsManager struct {
	mu       sync.Mutex
	registry map[ResourceTypeName][]*v1alpha1.Notification
}
