package globals

import (
	notifikv1alpha1 "freepik.com/notifik/api/v1alpha1"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
	"sync"
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
	var initialNotificationListState []*notifikv1alpha1.Notification

	initialStopSignalState := make(chan bool)
	initialMutexState := sync.Mutex{}

	Application.WatcherPool[watcherType] = ResourceTypeWatcherT{
		Mutex: &initialMutexState,

		Started: &initialStartedState,
		//Blocked:    &initialBlockedState,
		StopSignal: &initialStopSignalState,

		NotificationList: &initialNotificationListState,
	}
}

// TODO
func GetWatcherNotificationIndex(watcherType ResourceTypeName, notificationManifest *notifikv1alpha1.Notification) (result int) {

	notificationList := Application.WatcherPool[watcherType].NotificationList

	for notificationIndex, notification := range *notificationList {
		if (notification.Name == notificationManifest.Name) &&
			(notification.Namespace == notificationManifest.Namespace) {
			return notificationIndex
		}
	}

	return -1
}

// TODO
func GetWatcherPoolNotificationIndexes(notificationManifest *notifikv1alpha1.Notification) (result map[string]int) {

	result = make(map[string]int)

	for watcherType, _ := range Application.WatcherPool {
		notificationIndex := GetWatcherNotificationIndex(watcherType, notificationManifest)

		if notificationIndex != -1 {
			result[string(watcherType)] = notificationIndex
		}
	}

	return result
}

// TODO
func CreateWatcherNotification(watcherType ResourceTypeName, notificationManifest *notifikv1alpha1.Notification) {

	notificationList := Application.WatcherPool[watcherType].NotificationList

	(Application.WatcherPool[watcherType].Mutex).Lock()

	temporaryManifest := (*notificationManifest).DeepCopy()
	*notificationList = append(*notificationList, temporaryManifest)

	(Application.WatcherPool[watcherType].Mutex).Unlock()
}

// TODO
func DeleteWatcherNotificationByIndex(watcherType ResourceTypeName, notificationIndex int) {

	notificationList := Application.WatcherPool[watcherType].NotificationList

	(Application.WatcherPool[watcherType].Mutex).Lock()

	// Substitute the selected notification object with the last one from the list,
	// then replace the whole list with it, minus the last.
	//(*notificationList)[notificationIndex] = (*notificationList)[len(*notificationList)-1]
	//*notificationList = (*notificationList)[:len(*notificationList)-1]

	*notificationList = append((*notificationList)[:notificationIndex], (*notificationList)[notificationIndex+1:]...)

	(Application.WatcherPool[watcherType].Mutex).Unlock()
}
