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
	"os"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// TestSecretsCacheNeedsOnlyNamespacedRBAC proves against a real API server that, once restricted,
// the Secrets cache syncs with a namespaced Role, while the default cluster-wide one does not
func TestSecretsCacheNeedsOnlyNamespacedRBAC(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("KUBEBUILDER_ASSETS is not set: run it through 'make test'")
	}

	testEnv := &envtest.Environment{}
	adminConfig, err := testEnv.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testEnv.Stop() }()

	ctx := context.Background()
	adminClient, err := client.New(adminConfig, client.Options{Scheme: clientgoscheme.Scheme})
	if err != nil {
		t.Fatal(err)
	}

	// The controller can only read Secrets in 'team-a', the same as a namespaced Role does in a real cluster
	objects := []client.Object{
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "team-a"}},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "webhook-credentials", Namespace: "team-a"},
			StringData: map[string]string{"TOKEN": "s3cr3t"},
		},
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: "secrets-reader", Namespace: "team-a"},
			Rules: []rbacv1.PolicyRule{{
				APIGroups: []string{""}, Resources: []string{"secrets"}, Verbs: []string{"get", "list", "watch"},
			}},
		},
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "secrets-reader", Namespace: "team-a"},
			RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "Role", Name: "secrets-reader"},
			Subjects:   []rbacv1.Subject{{Kind: rbacv1.UserKind, APIGroup: rbacv1.GroupName, Name: "notifik"}},
		},
	}
	for _, object := range objects {
		if err := adminClient.Create(ctx, object); err != nil {
			t.Fatal(err)
		}
	}

	user, err := testEnv.AddUser(envtest.User{Name: "notifik"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Sanity check of the RBAC itself, without any cache: the user can list Secrets in 'team-a' only
	userClient, err := client.New(user.Config(), client.Options{Scheme: clientgoscheme.Scheme})
	if err != nil {
		t.Fatal(err)
	}
	if err := userClient.List(ctx, &corev1.SecretList{}, client.InNamespace("team-a")); err != nil {
		t.Fatalf("namespaced List() error = %v, want the Role to allow it", err)
	}
	if err := userClient.List(ctx, &corev1.SecretList{}); !apierrors.IsForbidden(err) {
		t.Fatalf("cluster-wide List() error = %v, want Forbidden", err)
	}

	tests := []struct {
		name       string
		namespaces []string
		timeout    time.Duration
		wantSynced bool
	}{
		{
			name:       "restricted cache syncs with a namespaced Role",
			namespaces: []string{"team-a"},
			timeout:    10 * time.Second,
			wantSynced: true,
		},
		{
			// The informer keeps retrying a forbidden cluster-wide List, so the Get gives up when the context ends
			name:       "default cluster-wide cache does not sync",
			namespaces: nil,
			timeout:    2 * time.Second,
			wantSynced: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secretsCache, err := cache.New(user.Config(), cache.Options{
				Scheme:   clientgoscheme.Scheme,
				ByObject: SecretsCacheByObject(tt.namespaces),
			})
			if err != nil {
				t.Fatal(err)
			}

			cacheCtx, cancel := context.WithTimeout(ctx, tt.timeout)
			defer cancel()
			go func() { _ = secretsCache.Start(cacheCtx) }()

			// Mimic the controller: Get a Secret through the cache, which starts its informer and
			// blocks until it syncs. Retry only while the cache itself is still starting
			secret := &corev1.Secret{}
			key := types.NamespacedName{Name: "webhook-credentials", Namespace: "team-a"}
			for {
				err = secretsCache.Get(cacheCtx, key, secret)
				if !errors.As(err, new(*cache.ErrCacheNotStarted)) || cacheCtx.Err() != nil {
					break
				}
				time.Sleep(50 * time.Millisecond)
			}

			if tt.wantSynced {
				if err != nil || string(secret.Data["TOKEN"]) != "s3cr3t" {
					t.Fatalf("Get() through the restricted cache error = %v, data = %v", err, secret.Data)
				}
				return
			}

			// controller-runtime reports an informer that never synced as a 504 Timeout status error
			if !apierrors.IsTimeout(err) || !strings.Contains(err.Error(), "Secret Informer to sync") {
				t.Fatalf("Get() through the cluster-wide cache error = %v, want a Secret informer sync timeout", err)
			}
		})
	}
}
