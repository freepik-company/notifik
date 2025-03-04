package integrations

import (
	"freepik.com/notifik/api/v1alpha1"
	"reflect"
)

// requestCredentials TODO
func requestCredentials(integration *v1alpha1.Integration) bool {

	if reflect.ValueOf(integration.Spec.Credentials).IsZero() {
		return false
	}

	if reflect.ValueOf(integration.Spec.Credentials.SecretRef).IsZero() {
		return false
	}

	return true
}
