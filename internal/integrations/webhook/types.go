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

package webhook

// TODO
type AlertmanagerAlertList []AlertmanagerAlert

// Alert represents the structure of an alert in Alertmanager
// Ref: https://prometheus.io/docs/alerting/latest/clients/
// Ref: https://raw.githubusercontent.com/prometheus/alertmanager/main/api/v2/openapi.yaml
type AlertmanagerAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`

	// Optional params
	StartsAt     string `json:"startsAt,omitempty"`
	EndsAt       string `json:"endsAt,omitempty"`
	GeneratorUrl string `json:"generatorURL,omitempty"`
}
