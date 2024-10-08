package webhook

import (
	"encoding/json"
	"fmt"
	"reflect"
)

const (
	amgrAlertDataUnmarshalErrorMessage         = "error decoding JSON from 'message.data' for Alertmanager validator: %s"
	amgrAlertDataRequiredStructureErrorMessage = "notification field 'message.data' does not meet the syntax requirements for Alertmanager: %s"
)

// ValidateAlertmanager checks whether the notification data meets the requirements for Alertmanager
func ValidateAlertmanager(data string) (result bool, hint string, err error) {

	alertList := AlertmanagerAlertList{AlertmanagerAlert{}}

	//
	err = json.Unmarshal([]byte(data), &alertList)
	if err != nil {
		return false, hint, fmt.Errorf(amgrAlertDataUnmarshalErrorMessage, err)
	}

	if reflect.ValueOf(alertList).IsZero() {
		return false, amgrAlertDataRequiredStructureErrorMessage, nil
	}

	//
	for _, alert := range alertList {

		// Check the main label for the alert.
		// This is required only on API v1
		_, alertNameFound := alert.Labels["alertname"]
		if !alertNameFound {
			hint = fmt.Sprintf("%s: %s", amgrAlertDataRequiredStructureErrorMessage, "label 'alertname' not found")
			return false, hint, nil
		}

		// Check whether startAt field is set
		if alert.StartsAt == "" {
			hint = fmt.Sprintf("%s: %s", amgrAlertDataRequiredStructureErrorMessage, "field 'startsAt' not found")
			return false, hint, nil
		}
	}

	return true, hint, nil
}
