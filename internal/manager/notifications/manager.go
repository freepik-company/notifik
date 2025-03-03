/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
