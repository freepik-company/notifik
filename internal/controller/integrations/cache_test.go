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
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"freepik.com/notifik/api/v1alpha1"
	integrationsRegistry "freepik.com/notifik/internal/registry/integrations"
)

func TestParseSecretsNamespaces(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "empty means all namespaces", raw: "", want: nil},
		{name: "only blanks means all namespaces", raw: " , ,", want: nil},
		{name: "single namespace", raw: "notifik", want: []string{"notifik"}},
		{name: "trimmed, deduplicated and sorted", raw: " team-b,team-a , team-b,,", want: []string{"team-a", "team-b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseSecretsNamespaces(tt.raw); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseSecretsNamespaces(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestSecretsCacheByObject(t *testing.T) {
	if got := SecretsCacheByObject(nil); got != nil {
		t.Errorf("SecretsCacheByObject(nil) = %v, want nil to keep the cluster-wide default", got)
	}

	got := SecretsCacheByObject([]string{"team-a", "team-b"})
	if len(got) != 1 {
		t.Fatalf("SecretsCacheByObject() returned %d objects, want only Secrets", len(got))
	}

	for object, byObject := range got {
		if _, ok := object.(*corev1.Secret); !ok {
			t.Errorf("SecretsCacheByObject() configures %T, want *corev1.Secret", object)
		}

		want := map[string]cache.Config{"team-a": {}, "team-b": {}}
		if !reflect.DeepEqual(byObject.Namespaces, want) {
			t.Errorf("SecretsCacheByObject() namespaces = %v, want %v", byObject.Namespaces, want)
		}
	}
}

func newIntegrationWithCredentials(secretNamespace string) *v1alpha1.Integration {
	return &v1alpha1.Integration{
		ObjectMeta: metav1.ObjectMeta{Name: "webhook-sender"},
		Spec: v1alpha1.IntegrationSpec{
			Type: "webhook",
			Credentials: v1alpha1.IntegrationCredentials{
				SecretRef: corev1.SecretReference{Name: "webhook-credentials", Namespace: secretNamespace},
			},
			Webhook: v1alpha1.IntegrationWebhook{
				Url:  "https://example.com/${TOKEN}",
				Verb: "POST",
			},
		},
	}
}

func newReconciler(t *testing.T, secretsNamespaces []string) *IntegrationReconciler {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "webhook-credentials", Namespace: "team-a"},
		Data:       map[string][]byte{"TOKEN": []byte("s3cr3t")},
	}

	return &IntegrationReconciler{
		Client:       fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build(),
		Scheme:       scheme,
		Options:      IntegrationControllerOptions{SecretsNamespaces: secretsNamespaces},
		Dependencies: IntegrationControllerDependencies{IntegrationsRegistry: integrationsRegistry.NewIntegrationsRegistry()},
	}
}

func TestReconcileIntegrationSecretsNamespaces(t *testing.T) {
	tests := []struct {
		name              string
		secretsNamespaces []string
		secretNamespace   string
		wantErr           string
	}{
		{name: "all namespaces by default", secretsNamespaces: nil, secretNamespace: "team-a"},
		{name: "secret in an allowed namespace", secretsNamespaces: []string{"team-a"}, secretNamespace: "team-a"},
		{
			name:              "secret outside the allowed namespaces",
			secretsNamespaces: []string{"team-b"},
			secretNamespace:   "team-a",
			wantErr:           "not in the list of allowed namespaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newReconciler(t, tt.secretsNamespaces)
			integration := newIntegrationWithCredentials(tt.secretNamespace)

			err := r.ReconcileIntegration(context.Background(), watch.Modified, integration)

			registered := r.Dependencies.IntegrationsRegistry.GetIntegrations()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("ReconcileIntegration() error = %v, want it to contain %q", err, tt.wantErr)
				}
				if len(registered) != 0 {
					t.Errorf("ReconcileIntegration() registered %d integrations on error, want 0", len(registered))
				}
				return
			}

			if err != nil {
				t.Fatalf("ReconcileIntegration() unexpected error = %v", err)
			}
			if len(registered) != 1 || registered[0].Spec.Webhook.Url != "https://example.com/s3cr3t" {
				t.Errorf("ReconcileIntegration() registered %+v, want one integration with expanded credentials", registered)
			}
		})
	}
}
