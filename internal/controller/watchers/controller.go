/*
Copyright 2024.

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

package watchers

import (
	"context"
	"fmt"
	"freepik.com/notifik/internal/integrations"
	"slices"
	"strings"
	"time"

	//
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	//
	"freepik.com/notifik/internal/globals"
	integrationsRegistry "freepik.com/notifik/internal/registry/integrations"
	notificationsRegistry "freepik.com/notifik/internal/registry/notifications"
	watchersRegistry "freepik.com/notifik/internal/registry/watchers"
	"freepik.com/notifik/internal/template"
)

const (
	// secondsToCheckWatcherAck is the number of seconds before checking
	// whether a watcher is started or not during watchers' reconciling process
	secondsToCheckWatcherAck = 10 * time.Second

	// secondsToReconcileWatchersAgain is the number of seconds to wait
	// between the moment of launching watchers, and repeating this process
	// (avoid the spam, mate)
	secondsToReconcileWatchersAgain = 2 * time.Second

	// secondsToResyncInformers is the number of seconds between syncing
	// all the manifests and repeating this process
	secondsToResyncInformers = 60 * 5 * time.Second

	//
	controllerContextFinishedMessage = "WatcherController finished by context"
	controllerWatcherStartedMessage  = "Watcher for '%s' has been started"
	controllerWatcherKilledMessage   = "Watcher for resource type '%s' killed by StopSignal"

	eventConditionsTriggerIntegrationsMessage = "Object has met conditions. Integrations will be triggered"

	watchedObjectParseError        = "Impossible to process watched object: %s"
	resourceWatcherLaunchingError  = "Impossible to start watcher for resource type: %s"
	integrationsSendMessageError   = "Impossible to send the message to some integration: %s"
	resourceWatcherGvrParsingError = "Failed to parse GVR from resourceType. Does it look like {group}/{version}/{resource}?"

	eventConditionGoTemplateError = "Go templating reported failure for object conditions: %s"
	eventMessageGoTemplateError   = "Go templating reported failure for object message: %s"
)

// WatchersControllerOptions represents available options that can be passed to WatchersController on start
type WatchersControllerOptions struct {
	// Duration to wait until resync all the objects
	InformerDurationToResync time.Duration
}

type WatchersControllerDependencies struct {
	Context *context.Context

	//
	IntegrationsRegistry  *integrationsRegistry.IntegrationsRegistry
	NotificationsRegistry *notificationsRegistry.NotificationsRegistry
	WatchersRegistry      *watchersRegistry.WatchersRegistry
}

// WatchersController represents the controller that triggers parallel threads.
// These threads process coming events against the conditions defined in Notification CRs
// Each thread is a watcher in charge of a group of resources GVRNN (Group + Version + Resource + Namespace + Name)
type WatchersController struct {
	Client client.Client

	Options      WatchersControllerOptions
	Dependencies WatchersControllerDependencies
}

// watchersCleanerWorker review the resource types of Notifications registry in the background.
// It disables the watchers that are not needed and delete them from watchers registry
// This function is intended to be used as goroutine
func (r *WatchersController) watchersCleanerWorker() {
	logger := log.FromContext(*r.Dependencies.Context)
	logger.Info("Starting watchers cleaner worker")

	for {
		//
		referentCandidates := r.Dependencies.NotificationsRegistry.GetRegisteredResourceTypes()
		evaluableCandidates := r.Dependencies.WatchersRegistry.GetRegisteredResourceTypes()

		//
		//logger.WithValues("types", referentCandidates).
		//	Debug("Current resource types in Notification registry")
		//logger.WithValues("types", evaluableCandidates).
		//	Debug("Current resource types in watchers registry")

		for _, resourceType := range evaluableCandidates {
			if !slices.Contains(referentCandidates, resourceType) {
				err := r.Dependencies.WatchersRegistry.DisableWatcher(resourceType)
				if err != nil {
					logger.WithValues("resourceType", resourceType).
						Info("Failed disabling watcher")
				}
			}
		}

		time.Sleep(5 * time.Second)
	}
}

// Start launches the WatchersController and keeps it alive
// It kills the controller on application's context death, and rerun the process when failed
func (r *WatchersController) Start() {
	logger := log.FromContext(*r.Dependencies.Context)

	// Start cleaner for dead watchers
	go r.watchersCleanerWorker()

	// Keep your controller alive
	for {
		select {
		case <-(*r.Dependencies.Context).Done():
			logger.Info(controllerContextFinishedMessage)
			return
		default:
			r.reconcileWatchers()
		}
	}
}

// reconcileWatchers checks each registered resource type and triggers watchers
// for those that are not already started.
func (r *WatchersController) reconcileWatchers() {
	logger := log.FromContext(*r.Dependencies.Context)

	for _, resourceType := range r.Dependencies.NotificationsRegistry.GetRegisteredResourceTypes() {

		_, watcherExists := r.Dependencies.WatchersRegistry.GetWatcher(resourceType)

		// Avoid wasting CPU for nothing
		if watcherExists && r.Dependencies.WatchersRegistry.IsStarted(resourceType) {
			continue
		}

		//
		if !watcherExists || !r.Dependencies.WatchersRegistry.IsStarted(resourceType) {
			go r.watchTypeWithInformer(resourceType)

			// Wait for the just started watcher to ACK itself
			time.Sleep(secondsToCheckWatcherAck)
			if !r.Dependencies.WatchersRegistry.IsStarted(resourceType) {
				logger.Info(fmt.Sprintf(resourceWatcherLaunchingError, resourceType))
			}
		}

		// Reduce CPU cycles spam
		time.Sleep(secondsToReconcileWatchersAgain)
	}
}

// watchTypeWithInformer creates and runs a Kubernetes informer for the specified
// resource type, and triggers processing for each event
func (r *WatchersController) watchTypeWithInformer(resourceType watchersRegistry.ResourceTypeName) {

	logger := log.FromContext(*r.Dependencies.Context)

	watcher, watcherExists := r.Dependencies.WatchersRegistry.GetWatcher(resourceType)
	if !watcherExists {
		watcher = r.Dependencies.WatchersRegistry.RegisterWatcher(resourceType)
	}

	logger.Info(fmt.Sprintf(controllerWatcherStartedMessage, resourceType))

	// Trigger ACK flag for watcher that is launching
	// Hey, this informer is blocking, so ACK is only disabled if the informer becomes dead
	_ = r.Dependencies.WatchersRegistry.SetStarted(resourceType, true)
	defer func() {
		_ = r.Dependencies.WatchersRegistry.SetStarted(resourceType, false)
	}()

	// Extract GVR + Namespace + Name from watched type:
	// {group}/{version}/{resource}/{namespace}/{name}
	GVRNN := strings.Split(resourceType, "/")
	if len(GVRNN) != 5 {
		logger.Info(resourceWatcherGvrParsingError)
		return
	}
	resourceGVR := schema.GroupVersionResource{
		Group:    GVRNN[0],
		Version:  GVRNN[1],
		Resource: GVRNN[2],
	}

	// Include the namespace when defined by the user (used as filter)
	namespace := corev1.NamespaceAll
	if GVRNN[3] != "" {
		namespace = GVRNN[3]
	}

	// Include the name when defined by the user (used as filter)
	name := GVRNN[4]

	var listOptionsFunc dynamicinformer.TweakListOptionsFunc = func(options *metav1.ListOptions) {}
	if name != "" {
		listOptionsFunc = func(options *metav1.ListOptions) {
			options.FieldSelector = "metadata.name=" + name
		}
	}

	// Listen to stop signal to kill this watcher just in case it's needed
	stopCh := make(chan struct{})

	go func() {
		<-watcher.StopSignal
		close(stopCh)
		logger.Info(fmt.Sprintf(controllerWatcherKilledMessage, resourceType))
	}()

	// Define our informer TODO
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(globals.Application.KubeRawClient,
		r.Options.InformerDurationToResync, namespace, listOptionsFunc)

	// Create an informer. This is a special type of client-go watcher that includes
	// mechanisms to hide disconnections, handle reconnections, and cache watched objects
	informer := factory.ForResource(resourceGVR).Informer()

	// Register functions to handle different types of events
	handlers := cache.ResourceEventHandlerFuncs{

		AddFunc: func(eventObject interface{}) {
			convertedEventObject := eventObject.(*unstructured.Unstructured)

			err := r.processEvent(resourceType, watch.Added, convertedEventObject.UnstructuredContent())
			if err != nil {
				logger.Error(err, fmt.Sprintf(watchedObjectParseError, err))
			}
		},
		UpdateFunc: func(eventObjectOld, eventObject interface{}) {
			convertedEventObjectOld := eventObjectOld.(*unstructured.Unstructured)
			convertedEventObject := eventObject.(*unstructured.Unstructured)

			err := r.processEvent(resourceType, watch.Modified,
				convertedEventObject.UnstructuredContent(), convertedEventObjectOld.UnstructuredContent())
			if err != nil {
				logger.Error(err, fmt.Sprintf(watchedObjectParseError, err))
			}
		},
		DeleteFunc: func(eventObject interface{}) {
			convertedEventObject := eventObject.(*unstructured.Unstructured)

			err := r.processEvent(resourceType, watch.Deleted, convertedEventObject.UnstructuredContent())
			if err != nil {
				logger.Error(err, fmt.Sprintf(watchedObjectParseError, err))
			}
		},
	}

	_, err := informer.AddEventHandler(handlers)
	if err != nil {
		logger.Error(err, "Error adding handling functions for events to an informer")
		return
	}

	informer.Run(stopCh)
}

// processEvent process an event coming from a watched resource type.
// It computes templating, evaluates conditions and decides whether to send a message for a given manifest
func (r *WatchersController) processEvent(resourceType watchersRegistry.ResourceTypeName, eventType watch.EventType, object ...map[string]interface{}) (err error) {
	logger := log.FromContext(*r.Dependencies.Context)

	notificationList := r.Dependencies.NotificationsRegistry.GetNotifications(resourceType)

	// Process only certain event types
	if eventType != watch.Added && eventType != watch.Modified && eventType != watch.Deleted {
		return nil
	}

	// Get object name and namespace for logging ease
	objectBasicData, err := GetObjectBasicData(&object[0])
	if err != nil {
		return err
	}

	// Create the object that will be injected on
	// Notification conditions/message on Golang template evaluation stage
	templateInjectedObject := map[string]interface{}{}

	templateInjectedObject["eventType"] = eventType
	templateInjectedObject["object"] = object[0]
	if eventType == watch.Modified {
		templateInjectedObject["previousObject"] = object[1]
	}

	//
	for _, notification := range notificationList {

		var conditionFlags []bool
		for _, condition := range notification.Spec.Conditions {
			parsedKey, err := template.EvaluateTemplate(condition.Key, templateInjectedObject)
			if err != nil {
				logger.WithValues(
					"notification", fmt.Sprintf("%s/%s", notification.Namespace, notification.Name),
					"object", fmt.Sprintf("%s/%s", objectBasicData["namespace"], objectBasicData["name"]),
					"error", err).Info(eventConditionGoTemplateError)
				conditionFlags = append(conditionFlags, false)
				continue
			}
			conditionFlags = append(conditionFlags, parsedKey == condition.Value)
		}

		if slices.Contains(conditionFlags, false) {
			continue
		}

		parsedMessage, err := template.EvaluateTemplate(notification.Spec.Message.Data, templateInjectedObject)
		if err != nil {
			logger.WithValues(
				"notification", fmt.Sprintf("%s/%s", notification.Namespace, notification.Name),
				"object", fmt.Sprintf("%s/%s", objectBasicData["namespace"], objectBasicData["name"]),
				"error", err).Info(eventMessageGoTemplateError)
			continue
		}

		logger.WithValues(
			"notification", fmt.Sprintf("%s/%s", notification.Namespace, notification.Name),
			"object", fmt.Sprintf("%s/%s", objectBasicData["namespace"], objectBasicData["name"])).
			Info(eventConditionsTriggerIntegrationsMessage)

		// Send the message through integrations
		err = integrations.SendMessage(*r.Dependencies.Context, r.Dependencies.IntegrationsRegistry,
			notification.Spec.Message.Integration.Name, parsedMessage)
		if err != nil {
			logger.WithValues(
				"notification", fmt.Sprintf("%s/%s", notification.Namespace, notification.Name),
				"object", fmt.Sprintf("%s/%s", objectBasicData["namespace"], objectBasicData["name"])).
				Info(fmt.Sprintf(integrationsSendMessageError, err))
		}
	}

	return err
}
