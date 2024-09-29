package webhook

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	//
	"freepik.com/notifik/api/v1alpha1"
)

const (
	//
	HttpEventPattern = `{"data":"%s","timestamp":"%s"}`

	//
	ValidatorNotFoundErrorMessage   = "validator %s not found"
	ValidationFailedErrorMessage    = "validation failed: %s"
	HttpRequestCreationErrorMessage = "error creating http request: %s"
	HttpRequestSendingErrorMessage  = "error sending http request: %s"
)

var (
	// validatorsMap is a map of integration names and their respective validation functions
	validatorsMap = map[string]func(data string) (result bool, hint string, err error){
		"alertmanager": ValidateAlertmanager,
	}
)

func SendMessage(ctx context.Context, params *v1alpha1.WebhookT, data string) (err error) {

	// Check if the webhook has a validator and execute it when available
	if params.Validator != "" {

		_, validatorFound := validatorsMap[params.Validator]
		if !validatorFound {
			return fmt.Errorf(ValidatorNotFoundErrorMessage, params.Validator)
		}

		//
		validatorResult, validatorHint, err := validatorsMap[params.Validator](data)
		if err != nil {
			return fmt.Errorf(ValidationFailedErrorMessage, err.Error())
		}

		if !validatorResult {
			return fmt.Errorf(ValidationFailedErrorMessage, validatorHint)
		}
	}

	httpClient := &http.Client{}

	// Create the request
	httpRequest, err := http.NewRequest(params.Verb, params.Url, nil)
	if err != nil {
		return fmt.Errorf(HttpRequestCreationErrorMessage, err)
	}

	// Add headers to the request
	for headerKey, headerValue := range params.Headers {
		httpRequest.Header.Set(headerKey, headerValue)
	}

	// Add data to the request
	payload := []byte(data)

	httpRequest.Body = io.NopCloser(bytes.NewBuffer(payload))
	httpRequest.Header.Set("Content-Type", "application/json")

	// Send HTTP request
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		return fmt.Errorf(HttpRequestSendingErrorMessage, err)
	}
	defer httpResponse.Body.Close()

	//
	return err
}
