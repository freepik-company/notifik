package alertmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"
	// corelog "log"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"freepik.com/notifik/internal/globals"
)

const (
	//
	HttpEventVerb                = "POST"
	HttpNotificationReasonHeader = "X-Notification-Reason"

	//
	HttpRequestCreationErrorMessage = "error creating http request: %s"
	HttpRequestSendingErrorMessage  = "error sending http request: %s"

	alertDataUnmarshalErrorMessage         = "error decoding JSON from Notification data: %s"
	alertListMarshalErrorMessage           = "error encoding alerts into JSON to be sent to Alertmanager: %s"
	alertDataRequiredStructureErrorMessage = "notification 'data' field does not meet the syntax requirements for Alertmanager"
)

// SendMessage TODO
// Ref: https://raw.githubusercontent.com/prometheus/alertmanager/main/api/v2/openapi.yaml
func SendMessage(ctx context.Context, reason string, data string) (err error) {
	logger := log.FromContext(ctx)
	_ = logger

	// 1. Check whether the data meets the requirements for Alertmanager
	alert := Alert{}
	err = json.Unmarshal([]byte(data), &alert)
	if err != nil {
		return errors.New(fmt.Sprintf(alertDataUnmarshalErrorMessage, err))
	}

	if reflect.ValueOf(alert).IsZero() {
		return errors.New(alertDataRequiredStructureErrorMessage)
	}

	// 2. Set the main label for the alert.
	// This is required only on API v1
	alert.Labels["alertname"] = reason

	if alert.StartAt == "" {
		currentMoment := time.Now().In(time.Local)
		alert.StartAt = currentMoment.Format(time.RFC3339)
	}

	// 3. Prepare the alert for being sent
	alertList := AlertList{alert}
	alertsJson, err := json.Marshal(alertList)
	if err != nil {
		return errors.New(fmt.Sprintf(alertListMarshalErrorMessage, err))
	}

	// TODO: DEBUG
	//corelog.Print(string(alertsJson))

	// 4. Perform HTTP request against Alertmanager
	httpClient := &http.Client{}

	alertmanagerConfig := globals.Application.Configuration.Integrations.Alertmanager

	// Create the request
	httpRequest, err := http.NewRequest(HttpEventVerb, alertmanagerConfig.Url, nil)
	if err != nil {
		return errors.New(fmt.Sprintf(HttpRequestCreationErrorMessage, err))
	}

	// Add headers to the request
	for headerKey, headerValue := range alertmanagerConfig.Headers {
		httpRequest.Header.Set(headerKey, headerValue)
	}

	httpRequest.Header.Set(HttpNotificationReasonHeader, reason)

	// Add data to the request
	httpRequest.Body = io.NopCloser(bytes.NewBuffer(alertsJson))
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
