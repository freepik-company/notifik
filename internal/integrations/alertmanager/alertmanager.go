package alertmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"freepik.com/jokati/internal/globals"
	"io"
	corelog "log"
	"net/http"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

const (
	HttpEventVerb = "POST"

	HttpRequestCreationErrorMessage = "error creating http request: %s"
	HttpRequestSendingErrorMessage  = "error sending http request: %s"
)

// SendMessage TODO
// Ref: https://raw.githubusercontent.com/prometheus/alertmanager/main/api/v2/openapi.yaml
func SendMessage(ctx context.Context, reason string, data string) (err error) {
	logger := log.FromContext(ctx)
	_ = logger

	corelog.Print("###########################################")
	corelog.Print(data)
	corelog.Print("###########################################")

	// 1. Check whether the data meets the requirements for Alertmanager
	alert := Alert{}
	err = json.Unmarshal([]byte(data), &alert)
	if err != nil {
		return errors.New(fmt.Sprintf("pepito: %s", err))
	}

	if reflect.ValueOf(alert).IsZero() {
		return errors.New("notification 'data' field does not meet the syntax requirements for Alertmanager")
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
		return errors.New(fmt.Sprintf("Cannot convert back alert into JSON: %s", err))
	}

	// TODO: DEBUG
	corelog.Print(string(alertsJson))

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

	httpRequest.Header.Set("X-Notification-Reason", reason)

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
