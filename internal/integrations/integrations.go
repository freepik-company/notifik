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
