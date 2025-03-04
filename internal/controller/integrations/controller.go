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

package integrations

import (
	"context"
	"fmt"
	//
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	//
	"freepik.com/notifik/api/v1alpha1"
	"freepik.com/notifik/internal/controller"
	"freepik.com/notifik/internal/registry/integrations"
)

type IntegrationControllerOptions struct{}

type IntegrationControllerDependencies struct {
	IntegrationsRegistry *integrations.IntegrationsRegistry
}

// IntegrationReconciler reconciles an Integration object
type IntegrationReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	//
	Options      IntegrationControllerOptions
	Dependencies IntegrationControllerDependencies
}

// +kubebuilder:rbac:groups=notifik.freepik.com,resources=integrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=notifik.freepik.com,resources=integrations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=notifik.freepik.com,resources=integrations/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *IntegrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	// 1. Get the content of the integration
	objectManifest := &v1alpha1.Integration{}
	err = r.Get(ctx, req.NamespacedName, objectManifest)

	// 2. Check the existence inside the cluster
	if err != nil {

		// 2.1 It does NOT exist: manage removal
		if err = client.IgnoreNotFound(err); err == nil {
			logger.Info(fmt.Sprintf(controller.ResourceNotFoundError, controller.IntegrationResourceType, req.Name))
			return result, err
		}

		// 2.2 Failed to get the resource, requeue the request
		logger.Info(fmt.Sprintf(controller.ResourceRetrievalError, controller.IntegrationResourceType, req.Name, err.Error()))
		return result, err
	}

	// 3. Check if the integration instance is marked to be deleted: indicated by the deletion timestamp being set
	if !objectManifest.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(objectManifest, controller.ResourceFinalizer) {
			// Delete Notification from WatcherPool
			err = r.ReconcileIntegration(ctx, watch.Deleted, objectManifest)
			if err != nil {
				logger.Info(fmt.Sprintf(controller.ResourceReconcileError, controller.IntegrationResourceType, req.Name, err.Error()))
				return result, err
			}

			// Remove the finalizers from Integration
			controllerutil.RemoveFinalizer(objectManifest, controller.ResourceFinalizer)
			err = r.Update(ctx, objectManifest)
			if err != nil {
				logger.Info(fmt.Sprintf(controller.ResourceFinalizersUpdateError, controller.IntegrationResourceType, req.Name, err.Error()))
			}
		}
		result = ctrl.Result{}
		err = nil
		return result, err
	}

	// 4. Add finalizer to the Integration
	if !controllerutil.ContainsFinalizer(objectManifest, controller.ResourceFinalizer) {
		controllerutil.AddFinalizer(objectManifest, controller.ResourceFinalizer)
		err = r.Update(ctx, objectManifest)
		if err != nil {
			return result, err
		}
	}

	// 5. Update the status before the requeue
	defer func() {
		err = r.Status().Update(ctx, objectManifest)
		if err != nil {
			logger.Info(fmt.Sprintf(controller.ResourceConditionUpdateError, controller.IntegrationResourceType, req.Name, err.Error()))
		}
	}()

	// 6. The Notification CR already exists: manage the update
	err = r.ReconcileIntegration(ctx, watch.Modified, objectManifest)
	if err != nil {
		r.UpdateConditionKubernetesApiCallFailure(objectManifest)
		logger.Info(fmt.Sprintf(controller.ResourceReconcileError, controller.IntegrationResourceType, req.Name, err.Error()))
		return result, err
	}

	// 7. Success, update the status
	r.UpdateConditionSuccess(objectManifest)

	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *IntegrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Watch Integrations
		For(&v1alpha1.Integration{}).
		Named("integration").

		// Watch Secrets and trigger reconciliation for Integrations using them
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
			requests := []reconcile.Request{}

			secret, ok := obj.(*corev1.Secret)
			if !ok {
				return []reconcile.Request{}
			}

			integrationList := r.Dependencies.IntegrationsRegistry.GetIntegrations()
			for _, integration := range integrationList {

				// Ignore integrations not asking for credentials
				if !requestCredentials(integration) {
					continue
				}

				//
				if integration.Spec.Credentials.SecretRef.Name == secret.Name &&
					integration.Spec.Credentials.SecretRef.Namespace == secret.Namespace {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      integration.Name,
							Namespace: integration.Namespace,
						},
					})
				}
			}

			return requests
		})).
		Complete(r)
}
