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

package xyz

import (
	"context"
	"fmt"
	"freepik.com/notifik/internal/integrations"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/dynamic"
	"slices"
	"strings"
	"time"

	//
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	//
	notifikv1alpha1 "freepik.com/notifik/api/v1alpha1"
	"freepik.com/notifik/internal/globals"
	"freepik.com/notifik/internal/template"
)

const (
	// secondsToCheckWatcherAck is the number of seconds before checking
	// whether a watcher is started or not during watchers' reconciling process
	secondsToCheckWatcherAck = 10 * time.Second

	// secondsBetweenWatchersReconcilingRetries is the number of seconds to wait between
	// loops in watchers' reconciling process (avoid the spam, mate)
	secondsBetweenWatchersReconcilingRetries = 2 * time.Second

	//
	controllerContextFinishedMessage         = "xyz.WorkloadController finished by context"
	controllerWatcherStartedMessage          = "Watcher for '%s' has been started"
	controllerWatcherKilledMessage           = "Watcher for resource type '%s' killed by StopSignal"
	controllerWatcherLaunchingBlockedMessage = "Watcher for '%s' is blocked and will not be started"

	eventConditionsTriggerIntegrationsMessage = "Object has met conditions. Integrations will be triggered"
	eventReceivedMessage                      = "Object event received"

	kubeWatcherStartFailedError    = "Impossible to watch resource type '%s'. RBAC issues?: %s"
	watchedObjectParseError        = "Impossible to process watched object: %s"
	runtimeObjectConversionError   = "Failed to parse object: %v"
	resourceWatcherLaunchingError  = "Impossible to start watcher for resource type: %s"
	integrationsSendMessageError   = "Impossible to send the message to some integration: %s"
	resourceWatcherGvrParsingError = "Failed to parse GVR from resourceType. Does it look like {group}/{version}/{resource}?"

	eventConditionGoTemplateError = "Go templating reported failure for object conditions: %s"
	eventMessageGoTemplateError   = "Go templating reported failure for object message: %s"
)

type WorkloadControllerOptions struct {
	// Events to be processed per second (best effort policy)
	EventsPerSecond int
}

// WorkloadController represents the controller that triggers parallel threads.
// These threads process coming events against the conditions defined in Notification CRs
// Each thread is a watcher in charge of a group of resources GVRNN (Group + Version + Resource + Namespace + Name)
type WorkloadController struct {
	Client client.Client

	//
	Options WorkloadControllerOptions
}

// Start launches the XYZ.WorkloadController and keeps it alive
// It kills the controller on application context death, and rerun the process when failed
func (r *WorkloadController) Start(ctx context.Context) {
	logger := log.FromContext(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.Info(controllerContextFinishedMessage)
			return
		default:
			r.reconcileWatchers(ctx)
		}
	}
}

// reconcileWatchers launches a parallel process that launches
// watchers for resource types defined into the WatcherPool
func (r *WorkloadController) reconcileWatchers(ctx context.Context) {
	logger := log.FromContext(ctx)

	for resourceType, resourceTypeWatcher := range globals.Application.WatcherPool.Pool {

		// TODO: Is this really needed or useful?
		// Check the existence of the resourceType into the WatcherPool.
		// Remember the controller.NotificationController can remove watchers on garbage collection
		if _, resourceTypeFound := globals.Application.WatcherPool.Pool[resourceType]; !resourceTypeFound {
			continue
		}

		// Prevent blocked watchers from being started.
		// Remember the controller.NotificationController blocks them during garbage collection
		if *resourceTypeWatcher.Blocked {
			continue
		}

		if !*resourceTypeWatcher.Started {
			go r.watchType(ctx, resourceType)

			// TODO: Improve following logic in future versions

			// Wait for the resourceType watcher to ACK itself into WatcherPool
			time.Sleep(secondsToCheckWatcherAck)
			if *(globals.Application.WatcherPool.Pool[resourceType].Started) == false {
				logger.Info(fmt.Sprintf(resourceWatcherLaunchingError, resourceType))
			}
		}

		// Wait a bit to reduce the spam to machine resources
		time.Sleep(secondsBetweenWatchersReconcilingRetries)
	}
}

// watchType launches a watcher for a certain resource type, and trigger processing for each entering resource event
func (r *WorkloadController) watchType(ctx context.Context, watchedType globals.ResourceTypeName) {

	logger := log.FromContext(ctx)

	logger.Info(fmt.Sprintf(controllerWatcherStartedMessage, watchedType))

	// Set ACK flag for watcher launching into the WatcherPool
	*(globals.Application.WatcherPool.Pool[watchedType].Started) = true
	defer func() {
		*(globals.Application.WatcherPool.Pool[watchedType].Started) = false
	}()

	notificationList := globals.Application.WatcherPool.Pool[watchedType].NotificationList

	// Extract GVR + Namespace + Name from watched type:
	// {group}/{version}/{resource}/{namespace}/{name}
	GVRNN := strings.Split(string(watchedType), "/")
	if len(GVRNN) != 5 {
		logger.Info(resourceWatcherGvrParsingError)
		return
	}
	resourceGVR := schema.GroupVersionResource{
		Group:    GVRNN[0],
		Version:  GVRNN[1],
		Resource: GVRNN[2],
	}

	namespace := GVRNN[3]
	name := GVRNN[4]

	//
	watchOptions := metav1.ListOptions{}

	// Include the name when defined by the user
	if name != "" {
		// DOCS: Alternative way to do the same
		// FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		watchOptions.FieldSelector = fields.OneTermEqualSelector(metav1.ObjectNameField, name).String()
	}

	// Include the namespace when defined by the user
	var resourceSelector dynamic.ResourceInterface
	resourceSelector = globals.Application.KubeRawClient.Resource(resourceGVR)
	if namespace != "" {
		resourceSelector = globals.Application.KubeRawClient.Resource(resourceGVR).Namespace(namespace)
	}

	// Create a watcher for defined resources
	resourceWatcher, err := resourceSelector.Watch(ctx, watchOptions)
	if err != nil {
		logger.Info(fmt.Sprintf(kubeWatcherStartFailedError, string(watchedType), err))
		return
	}
	defer resourceWatcher.Stop()

	// Listen to stop signal to kill this watcher just in case it's needed
	go func(p watch.Interface) {
		<-*(globals.Application.WatcherPool.Pool[watchedType].StopSignal)
		p.Stop()
		logger.Info(fmt.Sprintf(controllerWatcherKilledMessage, watchedType))
	}(resourceWatcher)

	// Calculate waiting time between loops to process N items per second
	// Done this way to allow limitation of consumed resources
	waitDuration := time.Second / time.Duration(r.Options.EventsPerSecond)

	//
	for WatchEvent := range resourceWatcher.ResultChan() {
		// Extract the unstructured object from the event
		objectMap, err := GetObjectMapFromRuntimeObject(&WatchEvent.Object)
		if err != nil {
			logger.Error(err, fmt.Sprintf(runtimeObjectConversionError, err))
			continue
		}

		// Process event for watched object apart
		// TODO: Probably we need to trigger processing into goroutines not to affect waiting calculation
		err = r.processEvent(ctx, notificationList, objectMap, WatchEvent.Type)
		if err != nil {
			logger.Error(err, fmt.Sprintf(watchedObjectParseError, err))
			continue
		}

		//
		time.Sleep(waitDuration)
	}
}

// processEvent process an event coming from a watched resource type.
// It computes templating, evaluates conditions and decides whether to send a message for a given manifest
func (r *WorkloadController) processEvent(ctx context.Context, notificationList *[]*notifikv1alpha1.Notification, object map[string]interface{}, eventType watch.EventType) (err error) {
	logger := log.FromContext(ctx)

	// Process only certain event types
	if eventType != watch.Added && eventType != watch.Modified && eventType != watch.Deleted {
		return nil
	}

	// Get object name and namespace for logging ease
	objectBasicData, err := GetObjectBasicData(&object)
	if err != nil {
		return err
	}

	// TODO: Show info about incoming object: GVRNN
	// TODO: Include these logs only under a flag
	// logger.WithValues(
	// 	"object", fmt.Sprintf("%s/%s", objectBasicData["namespace"], objectBasicData["name"])).
	// 	Info(eventReceivedMessage)

	for _, notification := range *notificationList {

		var conditionFlags []bool
		for _, condition := range notification.Spec.Conditions {
			parsedKey, err := template.EvaluateTemplate(condition.Key, object)
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

		parsedMessage, err := template.EvaluateTemplate(notification.Spec.Message.Data, object)
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
		err = integrations.SendMessage(ctx, notification.Spec.Message.Reason, parsedMessage)
		if err != nil {
			logger.Info(fmt.Sprintf(integrationsSendMessageError, err))
		}
	}

	return err
}
