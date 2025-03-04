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
	"freepik.com/notifik/api/v1alpha1"
)

func NewIntegrationsRegistry() *IntegrationsRegistry {
	return &IntegrationsRegistry{
		registry: []*v1alpha1.Integration{},
	}
}

// AddIntegration add an integration of provided type into registry
func (m *IntegrationsRegistry) AddIntegration(integration *v1alpha1.Integration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.registry = append(m.registry, integration)
}

// RemoveIntegration delete an integration of provided type
func (m *IntegrationsRegistry) RemoveIntegration(integration *v1alpha1.Integration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	integrations := m.registry
	index := -1
	for itemIndex, itemObject := range integrations {
		if itemObject.Name == integration.Name && itemObject.Namespace == integration.Namespace {
			index = itemIndex
			break
		}
	}
	if index != -1 {
		m.registry = append(integrations[:index], integrations[index+1:]...)
	}
}

// GetIntegrations return all the integrations
func (m *IntegrationsRegistry) GetIntegrations() []*v1alpha1.Integration {
	m.mu.Lock()
	defer m.mu.Unlock()

	//
	if len(m.registry) != 0 {
		return m.registry
	}

	return []*v1alpha1.Integration{}
}
