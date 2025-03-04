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
	"encoding/json"
	"errors"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"regexp"

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

var (
	ExpansionPatternRegex = regexp.MustCompile(`\$\{([^}]+)\}`)
)

// ReconcileIntegration keeps internal Integration resources' registry up-to-date
func (r *IntegrationReconciler) ReconcileIntegration(ctx context.Context, eventType watch.EventType, integrationManifest *v1alpha1.Integration) (err error) {
	logger := log.FromContext(ctx)

	// 1. Reject unknown events
	if eventType != watch.Modified && eventType != watch.Deleted {
		return nil
	}

	// 2. Handle 'delete' events
	if eventType == watch.Deleted {
		logger.Info(integrationDeletionMessage)
		r.Dependencies.IntegrationsRegistry.RemoveIntegration(integrationManifest)
		return nil
	}

	// 3. Handle 'create' / 'update' events
	logger.Info(integrationUpdatedMessage)
	defer func() {
		if err != nil {
			return
		}
		r.Dependencies.IntegrationsRegistry.RemoveIntegration(integrationManifest)
		r.Dependencies.IntegrationsRegistry.AddIntegration(integrationManifest)
	}()

	//
	if !requestCredentials(integrationManifest) {
		return nil
	}

	// Filled credentials must have name and namespace
	if requestCredentials(integrationManifest) &&
		(integrationManifest.Spec.Credentials.SecretRef.Name == "" ||
			integrationManifest.Spec.Credentials.SecretRef.Namespace == "") {
		return errors.New("integrations referencing credentials must have name and namespace")
	}

	//
	credentialsSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      integrationManifest.Spec.Credentials.SecretRef.Name,
		Namespace: integrationManifest.Spec.Credentials.SecretRef.Namespace,
	}, credentialsSecret)

	if err != nil {
		return errors.New(fmt.Sprintf("error fetching secret from Kubernetes: %v", err.Error()))
	}

	// Expand variables with values present in the secret
	varsExpandedIntegration, err := r.expandCredentials(integrationManifest, credentialsSecret)
	if err != nil {
		return errors.New(fmt.Sprintf("error expanding credentials: %v", err.Error()))
	}

	integrationManifest = varsExpandedIntegration
	return nil
}

// expandCredentials return a copy of passed Integration with ${expandable_patterns} already replaced
// with values from passed Secret
func (r *IntegrationReconciler) expandCredentials(integration *v1alpha1.Integration, secret *corev1.Secret) (result *v1alpha1.Integration, err error) {
	tmpIntegration := integration.DeepCopy()
	tmpIntegration.Spec.Credentials = v1alpha1.IntegrationCredentials{}

	// Convert object into JSON
	objectBytes, err := json.Marshal(tmpIntegration)
	if err != nil {
		return result, errors.New(fmt.Sprintf("error marshalling object to perform vars replacement: %v", err.Error()))
	}

	// Replace expandable variables by its actual value
	expandedString := ExpansionPatternRegex.ReplaceAllStringFunc(string(objectBytes), func(match string) string {
		// Extract variable name
		varName := match[2 : len(match)-1]

		//
		if replacement, exists := secret.Data[varName]; exists {
			return string(replacement)
		}
		return match
	})

	// Convert into an object again
	err = json.Unmarshal([]byte(expandedString), tmpIntegration)
	if err != nil {
		return result, errors.New(fmt.Sprintf("error unmarshalling object after vars replacement: %v", err.Error()))
	}

	// Restore the credentials
	tmpIntegration.Spec.Credentials = *integration.Spec.Credentials.DeepCopy()

	return tmpIntegration, nil
}
