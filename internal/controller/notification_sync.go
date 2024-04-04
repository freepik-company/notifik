package controller

import (
	"context"
	"strings"
	//
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/log"

	//
	notifikv1alpha1 "freepik.com/notifik/api/v1alpha1"
	"freepik.com/notifik/internal/globals"
)

const (

	// parseSyncTimeError error message for invalid value on 'synchronization' parameter
	parseSyncTimeError = "Can not parse the synchronization time from Notification: %s"

	//
	watcherPoolAddedNotificationMessage   = "A Notification will be added into the WatcherPool"
	watcherPoolUpdatedNotificationMessage = "A Notification will be modified into the WatcherPool"
	watcherPoolDeletedNotificationMessage = "A Notification will be deleted from WatcherPool"
)

// ReconcileNotification call Kubernetes API to actually Notification the resource
func (r *NotificationReconciler) ReconcileNotification(ctx context.Context, eventType watch.EventType, notificationManifest *notifikv1alpha1.Notification) (err error) {
	logger := log.FromContext(ctx)

	watchedTypeString := strings.Join([]string{
		notificationManifest.Spec.Watch.Group,
		notificationManifest.Spec.Watch.Version,
		notificationManifest.Spec.Watch.Resource,
		notificationManifest.Spec.Watch.Namespace,
		notificationManifest.Spec.Watch.Name,
	}, "/")
	watchedType := globals.ResourceTypeName(watchedTypeString)

	// Initialize the watcher into WatcherPool when not registered
	if _, watcherFound := globals.Application.WatcherPool.Pool[watchedType]; !watcherFound {
		globals.InitWatcher(watchedType)
	}

	notificationList := globals.Application.WatcherPool.Pool[watchedType].NotificationList
	//notificationIndex := globals.GetWatcherNotificationIndex(watchedType, notificationManifest)

	notificationIndexes := globals.GetWatcherPoolNotificationIndexes(notificationManifest)
	notificationIndex, notificationIndexFound := notificationIndexes[watchedTypeString]

	// Delete Notification from WatcherPool and exit
	if eventType == watch.Deleted {
		logger.Info(watcherPoolDeletedNotificationMessage,
			"watcher", watchedType)

		// Notification found, delete it from the pool
		if notificationIndexFound {
			globals.DeleteWatcherNotificationByIndex(watchedType, notificationIndex)
		}
		return nil
	}

	// Notification isn't found, create it into the pool
	if !notificationIndexFound {
		logger.Info(watcherPoolAddedNotificationMessage,
			"watcher", watchedType)

		globals.CreateWatcherNotification(watchedType, notificationManifest)

		// TODO: Decide if resourceType watcher restart is suitable on Notification creation events
		//*(globals.Application.WatcherPool[watchedType].StopSignal) <- true
		return nil
	}

	// Delete Notification from other Watchers when Notification is updated
	if eventType == watch.Modified {
		for currentWatchedType, currentNotificationIndex := range notificationIndexes {

			if currentWatchedType != watchedTypeString {
				globals.DeleteWatcherNotificationByIndex(globals.ResourceTypeName(currentWatchedType), currentNotificationIndex)
			}
		}
	}

	// Notification found, update it into the pool
	(*notificationList)[notificationIndex] = notificationManifest

	// TODO: Decide whether the cleaner should be executed by this controller or xyz.WorkloadController
	// Delete empty watcher from the WatcherPool.
	// This can be enabled setting a flag
	if r.Options.EnableWatcherPoolCleaner {
		globals.CleanWatcherPool()
	}

	// TODO: DELETE. DEBUG PURPOSES
	// corelog.Printf("notification controller loop checkpoint. event type: %v", eventType)

	return nil
}
