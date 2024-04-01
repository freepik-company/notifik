package webhook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	//
	"sigs.k8s.io/controller-runtime/pkg/log"

	//
	"freepik.com/notifik/internal/globals"
)

const (
	HttpEventPattern = `{"reason":"%s","data":"%s","timestamp":"%s"}`
	HttpEventVerb    = "POST"

	HttpRequestCreationErrorMessage = "error creating http request: %s"
	HttpRequestSendingErrorMessage  = "error sending http request: %s"
)

func SendMessage(ctx context.Context, reason string, data string) (err error) {
	logger := log.FromContext(ctx)
	_ = logger

	httpClient := &http.Client{}

	webhookConfig := globals.Application.Configuration.Integrations.Webhook

	// Create the request
	httpRequest, err := http.NewRequest(HttpEventVerb, webhookConfig.Url, nil)
	if err != nil {
		return errors.New(fmt.Sprintf(HttpRequestCreationErrorMessage, err))
	}

	// Add headers to the request
	for headerKey, headerValue := range webhookConfig.Headers {
		httpRequest.Header.Set(headerKey, headerValue)
	}

	httpRequest.Header.Set("X-Notification-Reason", reason)

	// Add data to the request
	payload := []byte(fmt.Sprintf(HttpEventPattern, reason, data, time.Now().In(time.Local)))
	httpRequest.Body = io.NopCloser(bytes.NewBuffer(payload))
	httpRequest.Header.Set("Content-Type", "application/json")

	// Send HTTP request
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		return errors.New(fmt.Sprintf(HttpRequestSendingErrorMessage, err))
	}
	defer httpResponse.Body.Close()

	//
	return err
}
