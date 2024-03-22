package controller

import (
	"context"
	"time"

	jokativ1alpha1 "freepik.com/jokati/api/v1alpha1"
)

const (

	// parseSyncTimeError error message for invalid value on 'synchronization' parameter
	parseSyncTimeError = "Can not parse the synchronization time from Notification: %s"

	// TODO
	ResourceKindDeployment = "Deployment"

	// TODO
	ActionDelete = "delete"
)

// GetSynchronizationTime return the spec.synchronization.time as duration, or default time on failures
func (r *NotificationReconciler) GetSynchronizationTime(notificationManifest *jokativ1alpha1.Notification) (synchronizationTime time.Duration, err error) {
	synchronizationTime, err = time.ParseDuration(notificationManifest.Spec.Synchronization.Time)
	if err != nil {
		err = NewErrorf(parseSyncTimeError, notificationManifest.Name)
		return synchronizationTime, err
	}

	return synchronizationTime, err
}

// reconcileNotification call Kubernetes API to actually Notification the resource
func (r *NotificationReconciler) reconcileNotification(ctx context.Context, notificationManifest *jokativ1alpha1.Notification) (err error) {

	return err
}
