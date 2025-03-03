/*
Copyright 2025.

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
