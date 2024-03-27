package globals

import (
	jokativ1alpha1 "freepik.com/jokati/api/v1alpha1"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
)

// NewKubernetesClient return a new Kubernetes Dynamic client from client-go SDK
func NewKubernetesClient() (client *dynamic.DynamicClient, err error) {
	config, err := ctrl.GetConfig()
	if err != nil {
		return client, err
	}

	// Create the clients to do requests to our friend: Kubernetes
	client, err = dynamic.NewForConfig(config)
	if err != nil {
		return client, err
	}

	return client, err
}

// CopyMap return a map that is a real copy of the original
// Ref: https://go.dev/blog/maps
func CopyMap(src map[string]interface{}) map[string]interface{} {
	m := make(map[string]interface{}, len(src))
	for k, v := range src {
		m[k] = v
	}
	return m
}

// SplitCommaSeparatedValues get a list of strings and return a new list
// where each element containing commas is divided in separated elements
func SplitCommaSeparatedValues(input []string) []string {
	var result []string
	for _, item := range input {
		parts := strings.Split(item, ",")
		result = append(result, parts...)
	}
	return result
}

// TODO
func InitWatcher(watcherType ResourceTypeName) {

	var initialStartedState bool = false
	var initialNotificationListState []*jokativ1alpha1.Notification

	initialStopSignalState := make(chan bool)

	Application.WatcherPool[watcherType] = ResourceTypeWatcherT{
		Started:          &initialStartedState,
		StopSignal:       &initialStopSignalState,
		NotificationList: &initialNotificationListState,
	}
}

// TODO
func GetWatcherNotificationIndex(watcherType ResourceTypeName, notificationManifest *jokativ1alpha1.Notification) (result int) {

	notificationList := Application.WatcherPool[watcherType].NotificationList

	for notificationIndex, notification := range *notificationList {
		if (notification.Name == notificationManifest.Name) &&
			(notification.Namespace == notificationManifest.Namespace) {
			return notificationIndex
		}
	}

	return -1
}
