/*
Copyright 2024.

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
	"freepik.com/notifik/api/v1alpha1"
	"golang.org/x/exp/maps"
	"reflect"
	"slices"
	"strings"
)

func NewNotificationsRegistry() *NotificationsRegistry {
	return &NotificationsRegistry{
		registry: make(map[ResourceTypeName][]*v1alpha1.Notification),
	}
}

// AddNotification add a notification of provided type into registry
func (m *NotificationsRegistry) AddNotification(rt ResourceTypeName, notification *v1alpha1.Notification) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// temporaryManifest := (*notificationManifest).DeepCopy()
	//	*notificationList = append(*notificationList, temporaryManifest)

	m.registry[rt] = append(m.registry[rt], notification)
}

// RemoveNotification delete a notification of provided type
func (m *NotificationsRegistry) RemoveNotification(rt ResourceTypeName, notification *v1alpha1.Notification) {
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
func (m *NotificationsRegistry) GetNotifications(rt ResourceTypeName) []*v1alpha1.Notification {
	m.mu.Lock()
	defer m.mu.Unlock()

	//
	if list, listFound := m.registry[rt]; listFound {
		return list
	}

	return []*v1alpha1.Notification{}
}

// GetRegisteredResourceTypes returns TODO
func (m *NotificationsRegistry) GetRegisteredResourceTypes() []ResourceTypeName {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Keys(m.registry)
}

// GetRegisteredExtraResourcesTypes returns TODO
func (m *NotificationsRegistry) GetRegisteredExtraResourcesTypes() []ResourceTypeName {

	m.mu.Lock()
	defer m.mu.Unlock()

	extraResourceTypes := []ResourceTypeName{}

	// Loop over all notifications collecting extra resources
	for _, resourceList := range m.registry {
		for _, notificationObj := range resourceList {
			for _, extraResource := range notificationObj.Spec.ExtraResources {

				// Prevents potential explosions due to
				// 'extraResource' comes empty from time to time
				if reflect.ValueOf(extraResource).IsZero() {
					continue
				}

				extraResourceName := strings.Join([]string{
					extraResource.Group, extraResource.Version, extraResource.Resource,
					extraResource.Namespace, extraResource.Name,
				}, "/")

				extraResourceTypes = append(extraResourceTypes, extraResourceName)
			}
		}
	}

	// Clean duplicated
	slices.Sort(extraResourceTypes)
	extraResourceTypes = slices.Compact(extraResourceTypes)

	return extraResourceTypes
}
