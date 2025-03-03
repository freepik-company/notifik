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
	"context"
	"strings"

	//
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/log"

	//
	"freepik.com/notifik/api/v1alpha1"
)

const (

	//
	notificationUpdatedMessage  = "A Notification was modified: will be updated into the internal registry"
	notificationDeletionMessage = "A Notification was deleted: will be deleted from internal registry"
)

// ReconcileNotification keeps internal Notification resources' registry up-to-date
func (r *NotificationReconciler) ReconcileNotification(ctx context.Context, eventType watch.EventType, notificationManifest *v1alpha1.Notification) (err error) {
	logger := log.FromContext(ctx)

	watchedType := strings.Join([]string{
		notificationManifest.Spec.Watch.Group,
		notificationManifest.Spec.Watch.Version,
		notificationManifest.Spec.Watch.Resource,
		notificationManifest.Spec.Watch.Namespace,
		notificationManifest.Spec.Watch.Name,
	}, "/")

	// Delete events
	if eventType == watch.Deleted {
		logger.Info(notificationDeletionMessage, "watcher", watchedType)

		r.Dependencies.NotificationsManager.RemoveNotification(watchedType, notificationManifest)
		return nil
	}

	// Create/Update events
	if eventType == watch.Modified {
		logger.Info(notificationUpdatedMessage, "watcher", watchedType)

		for _, registeredResourceType := range r.Dependencies.NotificationsManager.GetRegisteredResourceTypes() {
			r.Dependencies.NotificationsManager.RemoveNotification(registeredResourceType, notificationManifest)
		}

		r.Dependencies.NotificationsManager.AddNotification(watchedType, notificationManifest)
	}

	return nil
}
