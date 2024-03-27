package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	//
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/log"

	//
	jokativ1alpha1 "freepik.com/jokati/api/v1alpha1"
	"freepik.com/jokati/internal/globals"
)

const (

	// parseSyncTimeError error message for invalid value on 'synchronization' parameter
	parseSyncTimeError = "Can not parse the synchronization time from Notification: %s"

	//
	watcherPoolAddedNotificationMessage   = "A Notification will be added into the WatcherPool"
	watcherPoolUpdatedNotificationMessage = "A Notification will be modified into the WatcherPool"
	watcherPoolDeletedNotificationMessage = "A Notification will be deleted from WatcherPool"
)

// GetSynchronizationTime return the spec.synchronization.time as duration, or default time on failures
func (r *NotificationReconciler) GetSynchronizationTime(notificationManifest *jokativ1alpha1.Notification) (synchronizationTime time.Duration, err error) {
	synchronizationTime, err = time.ParseDuration(notificationManifest.Spec.Synchronization.Time)
	if err != nil {
		err = errors.New(fmt.Sprintf(parseSyncTimeError, notificationManifest.Name))
		return synchronizationTime, err
	}

	return synchronizationTime, err
}

// ReconcileNotification call Kubernetes API to actually Notification the resource
func (r *NotificationReconciler) ReconcileNotification(ctx context.Context, eventType watch.EventType, notificationManifest *jokativ1alpha1.Notification) (err error) {
	logger := log.FromContext(ctx)

	// TODO check if is the last watcher of his resource type in global map

	watchedTypeString := strings.Join([]string{
		notificationManifest.Spec.Watch.Group,
		notificationManifest.Spec.Watch.Version,
		notificationManifest.Spec.Watch.Resource,
	}, "/")
	watchedType := globals.ResourceTypeName(watchedTypeString)

	// Initialize the watcher into WatcherPool when not registered
	if _, watcherFound := globals.Application.WatcherPool[watchedType]; !watcherFound {
		globals.InitWatcher(watchedType)
	}

	notificationList := globals.Application.WatcherPool[watchedType].NotificationList
	notificationIndex := globals.GetWatcherNotificationIndex(watchedType, notificationManifest)

	// Delete Notification from WatcherPool and exit
	if eventType == watch.Deleted {
		logger.Info(watcherPoolDeletedNotificationMessage,
			"watcher", watchedType)

		// Notification found, delete it from the pool
		if notificationIndex != -1 {

			// Substitute the selected notification object with the last one from the list,
			// then replace the whole list with it, minus the last.
			(*notificationList)[notificationIndex] = (*notificationList)[len(*notificationList)-1]
			*notificationList = (*notificationList)[:len(*notificationList)-1]
		}
		return nil
	}

	// Notification isn't found, create it into the pool
	if notificationIndex == -1 {
		logger.Info(watcherPoolAddedNotificationMessage,
			"watcher", watchedType)
		*notificationList = append(*notificationList, notificationManifest)

		// TODO: Decide if resourceType watcher restart is suitable on Notification creation events
		//*(globals.Application.WatcherPool[watchedType].StopSignal) <- true
		return nil
	}

	// Notification found, update it into the pool
	// TODO: Decide if we want to log everything related to state
	//logger.Info(watcherPoolUpdatedNotificationMessage,
	//	"watcher", watchedType)
	(*notificationList)[notificationIndex] = notificationManifest

	// TODO: Decide if resourceType watcher restart is suitable on Notification update events
	//*(globals.Application.WatcherPool[watchedType].StopSignal) <- true
	return nil

}
