package notifications

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	//
	"freepik.com/notifik/api/v1alpha1"
)

const (
	// ConditionTypeResourceWatched indicates that the watcher for the resource type was launched or not
	ConditionTypeResourceWatched = "ResourceWatched"

	// Resource not found
	ConditionReasonResourceNotFound        = "ResourceNotFound"
	ConditionReasonResourceNotFoundMessage = "Resource was not found"

	// Template failed
	ConditionReasonInvalidTemplate        = "InvalidTemplate"
	ConditionReasonInvalidTemplateMessage = "Patch template is not valid. Deeper information inside the Patch status"

	// Failure
	ConditionReasonInvalidPatch        = "InvalidPatch"
	ConditionReasonInvalidPatchMessage = "Patch is invalid"

	// Success
	ConditionReasonResourceWatched        = "ResourceWatched"
	ConditionReasonResourceWatchedMessage = "Resource was successfully watched"

	////////////////////

	// ConditionTypeConditionsTemplateSucceed indicates that the templating stage was performed successfully for a condition
	ConditionTypeConditionsTemplateSucceed = "ConditionsTemplateSucceed"

	// Conditions template parsing failed
	ConditionReasonConditionsTemplateParsingFailed        = "ConditionsTemplateParsingFailed"
	ConditionReasonConditionsTemplateParsingFailedMessage = "Golang returned: %s"

	// Success
	ConditionReasonConditionsTemplateParsed        = "ConditionsTemplateParsed"
	ConditionReasonConditionsTemplateParsedMessage = "Conditions template was successfully parsed"

	////////////////////

	// ConditionTypeMessageTemplateSucceed indicates that the templating stage was performed successfully for the message
	ConditionTypeMessageTemplateSucceed = "MessageTemplateSucceed"

	// Message template parsing failed
	ConditionReasonMessageTemplateParsingFailed        = "MessageTemplateParsingFailed"
	ConditionReasonMessageTemplateParsingFailedMessage = "Golang returned: %s"

	// Success
	ConditionReasonMessageTemplateParsed        = "MessageTemplateParsed"
	ConditionReasonMessageTemplateParsedMessage = "Message template was successfully parsed"
)

// NewNotificationCondition a set of default options for creating a Condition.
func NewNotificationCondition(condType string, status metav1.ConditionStatus, reason, message string) *metav1.Condition {
	return &metav1.Condition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// GetNotificationCondition returns the condition with the provided type.
func GetNotificationCondition(patch *v1alpha1.Notification, condType string) *metav1.Condition {

	for i, v := range patch.Status.Conditions {
		if v.Type == condType {
			return &patch.Status.Conditions[i]
		}
	}
	return nil
}

// UpdateNotificationCondition update or create a new condition inside the status of the CR
func UpdateNotificationCondition(patch *v1alpha1.Notification, condition *metav1.Condition) {

	// Get the condition
	currentCondition := GetNotificationCondition(patch, condition.Type)

	if currentCondition == nil {
		// Create the condition when not existent
		patch.Status.Conditions = append(patch.Status.Conditions, *condition)
	} else {
		// Update the condition when existent.
		currentCondition.Status = condition.Status
		currentCondition.Reason = condition.Reason
		currentCondition.Message = condition.Message
		currentCondition.LastTransitionTime = metav1.Now()
	}
}
