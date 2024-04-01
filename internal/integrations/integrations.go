package integrations

import (
	"context"
	"reflect"

	//
	"freepik.com/notifik/internal/globals"
	"freepik.com/notifik/internal/integrations/alertmanager"
	"freepik.com/notifik/internal/integrations/webhook"
)

// SendMessage send a message to all configured integrations
func SendMessage(ctx context.Context, reason string, msg string) (err error) {

	// Send the message to Alertmanager
	if !reflect.ValueOf(globals.Application.Configuration.Integrations.Alertmanager).IsZero() {
		err = alertmanager.SendMessage(ctx, reason, msg)
		if err != nil {
			return err
		}
	}

	// Send the message to a webhook
	if !reflect.ValueOf(globals.Application.Configuration.Integrations.Webhook).IsZero() {
		err = webhook.SendMessage(ctx, reason, msg)
		if err != nil {
			return err
		}
	}

	return nil
}
