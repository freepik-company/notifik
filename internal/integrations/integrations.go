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
	"reflect"

	//
	"freepik.com/notifik/internal/globals"
	"freepik.com/notifik/internal/integrations/webhook"
)

// SendMessage send a message to a specific integration
func SendMessage(ctx context.Context, integrationName string, msg string) (err error) {

	//
	integrationFound := false
	for _, integObj := range globals.Application.Configuration.Integrations {

		if integObj.Name != integrationName {
			continue
		}
		integrationFound = true

		//
		if integObj.Type == "webhook" {

			// TODO: Perform this check on config initialization, not here
			if reflect.ValueOf(integObj.Webhook).IsZero() {
				return fmt.Errorf("webhook configuration missing for integration %s", integrationName)
			}

			err = webhook.SendMessage(ctx, &integObj.Webhook, msg)
			if err != nil {
				return err
			}
		}

		// Implement other integrations here
		////////////////////////////////////

	}

	if !integrationFound {
		return fmt.Errorf("integration '%s' not found", integrationName)
	}

	return nil
}
