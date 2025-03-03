package notifications

import (
	v1alpha1 "freepik.com/notifik/api/v1alpha1"
	"golang.org/x/exp/maps"
)

func NewNotificationsManager() *NotificationsManager {
	return &NotificationsManager{
		registry: make(map[ResourceTypeName][]*v1alpha1.Notification),
	}
}

// AddNotification add a notification of provided type into registry
func (m *NotificationsManager) AddNotification(rt ResourceTypeName, notification *v1alpha1.Notification) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// temporaryManifest := (*notificationManifest).DeepCopy()
	//	*notificationList = append(*notificationList, temporaryManifest)

	m.registry[rt] = append(m.registry[rt], notification)
}

// RemoveNotification delete a notification of provided type
func (m *NotificationsManager) RemoveNotification(rt ResourceTypeName, notification *v1alpha1.Notification) {
	m.mu.Lock()
	defer m.mu.Unlock()

	notifications := m.registry[rt]
	index := -1
	for itemIndex, itemObject := range notifications {
		if itemObject.Name == notification.Name && itemObject.Namespace == notification.Namespace {
			index = itemIndex
			break
		}
	}
	if index != -1 {
		m.registry[rt] = append(notifications[:index], notifications[index+1:]...)
	}

	// Delete index from registry when any Notification resource is needing it
	if len(m.registry[rt]) == 0 {
		delete(m.registry, rt)
	}
}

// GetNotifications return all the notifications of provided type
func (m *NotificationsManager) GetNotifications(rt ResourceTypeName) []*v1alpha1.Notification {
	m.mu.Lock()
	defer m.mu.Unlock()

	//
	if list, listFound := m.registry[rt]; listFound {
		return list
	}

	return []*v1alpha1.Notification{}
}

// GetRegisteredResourceTypes returns TODO
func (m *NotificationsManager) GetRegisteredResourceTypes() []ResourceTypeName {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Keys(m.registry)
}
