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
	"errors"
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"freepik.com/notifik/api/v1alpha1"
	"freepik.com/notifik/internal/controller"
	integrationsRegistry "freepik.com/notifik/internal/registry/integrations"
)

func TestParseSecretsNamespaces(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    []string
		wantErr bool
	}{
		{name: "empty means all namespaces", raw: "", want: nil},
		{name: "only blanks means all namespaces", raw: " , ,", want: nil},
		{name: "single namespace", raw: "notifik", want: []string{"notifik"}},
		{name: "trimmed, deduplicated and sorted", raw: " team-b,team-a , team-b,,", want: []string{"team-a", "team-b"}},
		{name: "wildcard is rejected", raw: "*", wantErr: true},
		{name: "uppercase is rejected", raw: "notifik,Team-A", wantErr: true},
		{name: "inner spaces are rejected", raw: "team a", wantErr: true},
		{name: "dots are rejected", raw: "team.a", wantErr: true},
		{name: "too long is rejected", raw: strings.Repeat("a", 64), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSecretsNamespaces(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseSecretsNamespaces(%q) = %v, want an error", tt.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSecretsNamespaces(%q) unexpected error = %v", tt.raw, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
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

func newReconciler(t *testing.T, secretsNamespaces []string, objects ...client.Object) *IntegrationReconciler {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "webhook-credentials", Namespace: "team-a"},
		Data:       map[string][]byte{"TOKEN": []byte("s3cr3t")},
	}

	return &IntegrationReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).
			WithObjects(append(objects, secret)...).
			WithStatusSubresource(&v1alpha1.Integration{}).
			Build(),
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
				if !errors.Is(err, reconcile.TerminalError(nil)) {
					t.Errorf("ReconcileIntegration() error = %v, want a terminal error", err)
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

// An Integration that was working and is edited to point at a Secret outside the allowed namespaces
// must be unregistered, so it never keeps sending with the credentials expanded before the edit
func TestReconcileIntegrationEditedToDisallowedNamespace(t *testing.T) {
	r := newReconciler(t, []string{"team-a"})

	if err := r.ReconcileIntegration(context.Background(), watch.Modified, newIntegrationWithCredentials("team-a")); err != nil {
		t.Fatalf("ReconcileIntegration() unexpected error = %v", err)
	}
	if got := r.Dependencies.IntegrationsRegistry.GetIntegrations(); len(got) != 1 {
		t.Fatalf("ReconcileIntegration() registered %d integrations, want 1", len(got))
	}

	err := r.ReconcileIntegration(context.Background(), watch.Modified, newIntegrationWithCredentials("team-b"))
	if !errors.Is(err, reconcile.TerminalError(nil)) {
		t.Fatalf("ReconcileIntegration() error = %v, want a terminal error", err)
	}
	if got := r.Dependencies.IntegrationsRegistry.GetIntegrations(); len(got) != 0 {
		t.Errorf("ReconcileIntegration() kept %d integrations registered, want the stale one removed", len(got))
	}
}

// The terminal error must reach controller-runtime through Reconcile, despite the deferred status update
func TestReconcileReturnsTerminalErrorForDisallowedNamespace(t *testing.T) {
	integration := newIntegrationWithCredentials("team-a")
	r := newReconciler(t, []string{"team-b"}, integration)

	request := ctrl.Request{NamespacedName: types.NamespacedName{Name: integration.Name}}
	_, err := r.Reconcile(context.Background(), request)
	if !errors.Is(err, reconcile.TerminalError(nil)) {
		t.Fatalf("Reconcile() error = %v, want a terminal error", err)
	}
	if !strings.Contains(err.Error(), "not in the list of allowed namespaces") {
		t.Errorf("Reconcile() error = %v, want the allowed namespaces message", err)
	}
}

// Credentials without name or namespace can only be fixed by editing the Integration: do not requeue it
func TestReconcileIntegrationIncompleteSecretRefIsTerminal(t *testing.T) {
	r := newReconciler(t, nil)
	integration := newIntegrationWithCredentials("")

	err := r.ReconcileIntegration(context.Background(), watch.Modified, integration)
	if !errors.Is(err, reconcile.TerminalError(nil)) {
		t.Fatalf("ReconcileIntegration() error = %v, want a terminal error", err)
	}
	if !strings.Contains(err.Error(), "must have name and namespace") {
		t.Errorf("ReconcileIntegration() error = %v, want the name and namespace message", err)
	}
}

// The status must say why the Integration is not synced, instead of a generic Kubernetes API failure
func TestReconcileSetsSecretNamespaceNotAllowedCondition(t *testing.T) {
	integration := newIntegrationWithCredentials("team-a")
	r := newReconciler(t, []string{"team-b"}, integration)

	request := ctrl.Request{NamespacedName: types.NamespacedName{Name: integration.Name}}
	_, err := r.Reconcile(context.Background(), request)
	if !errors.Is(err, ErrSecretNamespaceNotAllowed) || !errors.Is(err, reconcile.TerminalError(nil)) {
		t.Fatalf("Reconcile() error = %v, want a terminal ErrSecretNamespaceNotAllowed", err)
	}

	got := &v1alpha1.Integration{}
	if err := r.Get(context.Background(), request.NamespacedName, got); err != nil {
		t.Fatal(err)
	}
	condition := meta.FindStatusCondition(got.Status.Conditions, controller.ConditionTypeResourceSynced)
	switch {
	case condition == nil:
		t.Errorf("Reconcile() set conditions %+v, want %s", got.Status.Conditions, controller.ConditionTypeResourceSynced)
	case condition.Reason != controller.ConditionReasonSecretNamespaceNotAllowed || condition.Status != metav1.ConditionFalse:
		t.Errorf("Reconcile() condition = %s/%s, want %s/%s", condition.Status, condition.Reason,
			metav1.ConditionFalse, controller.ConditionReasonSecretNamespaceNotAllowed)
	}
}
