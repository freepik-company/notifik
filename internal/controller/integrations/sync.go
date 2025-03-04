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
	//
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/log"

	//
	"freepik.com/notifik/api/v1alpha1"
)

const (

	//
	integrationUpdatedMessage  = "An Integration was modified: will be updated into the internal registry"
	integrationDeletionMessage = "An Integration was deleted: will be deleted from internal registry"
)

// ReconcileIntegration keeps internal Integration resources' registry up-to-date
func (r *IntegrationReconciler) ReconcileIntegration(ctx context.Context, eventType watch.EventType, integrationManifest *v1alpha1.Integration) (err error) {
	logger := log.FromContext(ctx)

	// Delete events
	if eventType == watch.Deleted {
		logger.Info(integrationDeletionMessage)

		r.Dependencies.IntegrationsRegistry.RemoveIntegration(integrationManifest)
		return nil
	}

	// Create/Update events
	if eventType == watch.Modified {
		logger.Info(integrationUpdatedMessage)

		// If it's asking for a secret, retrieve it

		//// Ignore integrations not asking for credentials
		//if reflect.ValueOf(integrationManifest.Spec.Credentials).IsZero() {
		//	continue
		//}
		//
		//if reflect.ValueOf(integration.Spec.Credentials.SecretRef).IsZero() {
		//	continue
		//}
		//
		//if integration.Spec.Credentials.SecretRef.Name == secret.Name &&
		//	integration.Spec.Credentials.SecretRef.Namespace == secret.Namespace {
		//	requests = append(requests, reconcile.Request{
		//		NamespacedName: types.NamespacedName{
		//			Name:      integration.Name,
		//			Namespace: integration.Namespace,
		//		},
		//	})
		//}

		// Expand variables present in the secret
		// Store the integration

		r.Dependencies.IntegrationsRegistry.RemoveIntegration(integrationManifest)
		r.Dependencies.IntegrationsRegistry.AddIntegration(integrationManifest)
	}

	return nil
}
