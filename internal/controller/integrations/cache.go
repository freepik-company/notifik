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
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ParseSecretsNamespaces turns a comma-separated list of namespaces into a sorted slice
// without blanks nor duplicates. An empty input returns nil, which means 'all namespaces'.
// Every namespace must be a valid DNS-1123 label, so typos or wildcards fail fast
func ParseSecretsNamespaces(raw string) ([]string, error) {
	items := strings.Split(raw, ",")
	namespaces := make([]string, 0, len(items))

	for _, namespace := range items {
		namespace = strings.TrimSpace(namespace)
		if namespace == "" || slices.Contains(namespaces, namespace) {
			continue
		}
		if errs := validation.IsDNS1123Label(namespace); len(errs) > 0 {
			return nil, fmt.Errorf("invalid namespace '%s': %s", namespace, strings.Join(errs, "; "))
		}
		namespaces = append(namespaces, namespace)
	}

	if len(namespaces) == 0 {
		return nil, nil
	}

	slices.Sort(namespaces)
	return namespaces, nil
}

// SecretsCacheByObject returns the cache configuration that restricts the Secret informer,
// used to read Integration credentials, to the given namespaces.
// An empty list returns nil, keeping the default behaviour: Secrets are cached cluster-wide
func SecretsCacheByObject(namespaces []string) map[client.Object]cache.ByObject {
	if len(namespaces) == 0 {
		return nil
	}

	namespacesConfig := make(map[string]cache.Config, len(namespaces))
	for _, namespace := range namespaces {
		namespacesConfig[namespace] = cache.Config{}
	}

	return map[client.Object]cache.ByObject{
		&corev1.Secret{}: {Namespaces: namespacesConfig},
	}
}

// secretsNamespaceAllowed returns whether the controller is able to read Secrets from the given namespace
func (o *IntegrationControllerOptions) secretsNamespaceAllowed(namespace string) bool {
	if len(o.SecretsNamespaces) == 0 {
		return true
	}

	return slices.Contains(o.SecretsNamespaces, namespace)
}
