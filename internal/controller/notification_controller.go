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

package controller

import (
	"context"
	"fmt"
	"time"

	//
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	//
	"freepik.com/notifik/api/v1alpha1"
)

const (
	defaultSyncTimeForExitWithError = 10 * time.Second
	notificationFinalizer           = "notifik.freepik.com/finalizer"

	scheduleSynchronization = "Schedule synchronization in: %s"

	notificationNotFoundError          = "Notification resource not found. Ignoring since object must be deleted."
	notificationRetrievalError         = "Error getting the notification from the cluster"
	notificationFinalizersUpdateError  = "Failed to update finalizer of notification: %s"
	notificationConditionUpdateError   = "Failed to update the condition on notification: %s"
	notificationSyncTimeRetrievalError = "Can not get synchronization time from the notification: %s"
	notificationReconcileError         = "Can not reconcile Notification: %s"
)

type NotificationControllerOptions struct {

	// Enable WatcherPool cleaner process
	EnableWatcherPoolCleaner bool
}

// NotificationReconciler reconciles a Notification object
type NotificationReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	//
	Options NotificationControllerOptions
}

//+kubebuilder:rbac:groups=notifik.freepik.com,resources=notifications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=notifik.freepik.com,resources=notifications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=notifik.freepik.com,resources=notifications/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets;configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *NotificationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)
	_ = logger

	// 1. Get the content of the notification
	notificationManifest := &v1alpha1.Notification{}
	err = r.Get(ctx, req.NamespacedName, notificationManifest)

	// 2. Check the existence inside the cluster
	if err != nil {

		// 2.1 It does NOT exist: manage removal
		if err = client.IgnoreNotFound(err); err == nil {
			logger.Info(notificationNotFoundError)
			return result, err
		}

		// 2.2 Failed to get the resource, requeue the request
		logger.Info(notificationRetrievalError)
		return result, err
	}

	// 3. Check if the notification instance is marked to be deleted: indicated by the deletion timestamp being set
	if !notificationManifest.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(notificationManifest, notificationFinalizer) {
			// Delete Notification from WatcherPool
			err = r.ReconcileNotification(ctx, watch.Deleted, notificationManifest)
			if err != nil {
				logger.Info(fmt.Sprintf(notificationReconcileError, notificationManifest.Name))
				return result, err
			}

			// Remove the finalizers on notification CR
			controllerutil.RemoveFinalizer(notificationManifest, notificationFinalizer)
			err = r.Update(ctx, notificationManifest)
			if err != nil {
				logger.Info(fmt.Sprintf(notificationFinalizersUpdateError, req.Name))
			}
		}
		result = ctrl.Result{}
		err = nil
		return result, err
	}

	// 4. Add finalizer to the notification CR
	if !controllerutil.ContainsFinalizer(notificationManifest, notificationFinalizer) {
		controllerutil.AddFinalizer(notificationManifest, notificationFinalizer)
		err = r.Update(ctx, notificationManifest)
		if err != nil {
			return result, err
		}
	}

	// 5. Update the status before the requeue
	defer func() {
		err = r.Status().Update(ctx, notificationManifest)
		if err != nil {
			logger.Info(fmt.Sprintf(notificationConditionUpdateError, req.Name))
		}
	}()

	// 6. Schedule periodical request
	//RequeueTime, err := r.GetSynchronizationTime(notificationManifest)
	//if err != nil {
	//	logger.Info(fmt.Sprintf(notificationSyncTimeRetrievalError, notificationManifest.Name))
	//	return result, err
	//}
	//result = ctrl.Result{
	//	RequeueAfter: RequeueTime,
	//}

	// 7. The Notification CR already exists: manage the update
	err = r.ReconcileNotification(ctx, watch.Modified, notificationManifest)
	if err != nil {
		logger.Info(fmt.Sprintf(notificationReconcileError, notificationManifest.Name))
		return result, err
	}

	// 8. Success, update the status
	UpdateNotificationCondition(notificationManifest, NewNotificationCondition(ConditionTypeResourceWatched,
		metav1.ConditionTrue,
		ConditionReasonResourceWatched,
		ConditionReasonResourceWatchedMessage,
	))

	//logger.Info(fmt.Sprintf(scheduleSynchronization, result.RequeueAfter.String()))
	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *NotificationReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Notification{}).
		Complete(r)
}
