package controller

const (
	ResourceFinalizer = "notifik.freepik.com/finalizer"

	NotificationResourceType = "Notification"
	IntegrationResourceType  = "Integration"

	//
	ResourceNotFoundError         = "%s '%s' resource not found. Ignoring since object must be deleted."
	ResourceRetrievalError        = "Error getting the %s '%s' from the cluster: %s"
	ResourceFinalizersUpdateError = "Failed to update finalizer of %s '%s': %s"
	ResourceConditionUpdateError  = "Failed to update the condition on %s '%s': %s"
	ResourceReconcileError        = "Can not reconcile %s '%s': %s"
)
