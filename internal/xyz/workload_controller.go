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
	jokativ1alpha1 "freepik.com/jokati/api/v1alpha1"
	"freepik.com/jokati/internal/globals"
	"freepik.com/jokati/internal/template"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"slices"
	"strings"
	"time"
)

const (
	secondsBetweenWatchingRetries = 10 * time.Second

	controllerContextFinishedMessage = "xyz.WorkloadController finished by context"
	controllerWatcherStartedMessage  = "Watcher for '%s' has been started"

	kubeWatcherStartFailedError   = "Impossible to watch resource type '%s'. RBAC issues?: %s"
	watchedObjectParseError       = "Impossible to process watched object: %s"
	runtimeObjectConversionError  = "Failed to parse object: %v"
	resourceWatcherLaunchingError = "Impossible to start watcher for resource type: %s"
)

// WorkloadController TODO
type WorkloadController struct {
	Client client.Client
}

// TODO
func (r *WorkloadController) Start(ctx context.Context) {
	logger := log.FromContext(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.Info(controllerContextFinishedMessage)
			return
		default:
			r.ReconcileWatchers(ctx)
			// TODO: Do we need to wait?
		}
	}
}

func (r *WorkloadController) ReconcileWatchers(ctx context.Context) {
	logger := log.FromContext(ctx)

	for resourceType, resourceTypeWatcher := range globals.Application.WatcherPool {

		if !*resourceTypeWatcher.Started {
			go r.WatchType(ctx, resourceType)

			// TODO: Explotó, y yo exploté de él
			time.Sleep(secondsBetweenWatchingRetries)
			if *(globals.Application.WatcherPool[resourceType].Started) == false {
				logger.Info(fmt.Sprintf(resourceWatcherLaunchingError, resourceType))
			}
		}
	}
}

// WatchType TODO
func (r *WorkloadController) WatchType(ctx context.Context, watchedType globals.ResourceTypeName) {

	logger := log.FromContext(ctx)

	logger.Info(fmt.Sprintf(controllerWatcherStartedMessage, watchedType))

	// TODO: METER A FALSE EL FLAG DE RUNNING DE ESTA GOROUTINE
	// Set ACK flag for watcher launching into the WatcherPool
	*(globals.Application.WatcherPool[watchedType].Started) = true
	defer func() {
		*(globals.Application.WatcherPool[watchedType].Started) = false
	}()

	notificationList := globals.Application.WatcherPool[watchedType].NotificationList

	// Extract GVR from watched type:
	// {group}/{version}/{resource}
	GVR := strings.Split(string(watchedType), "/")
	if len(GVR) != 3 {
		// TODO breaking the law
	}
	resourceGVR := schema.GroupVersionResource{
		Group:    GVR[0],
		Version:  GVR[1],
		Resource: GVR[2],
	}

	// Create a watcher for defined resources
	resourceWatcher, err := globals.Application.KubeRawClient.Resource(resourceGVR).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Info(fmt.Sprintf(kubeWatcherStartFailedError, string(watchedType), err))
		return
	}
	defer resourceWatcher.Stop()

	for WatchEvent := range resourceWatcher.ResultChan() {
		// Extract the unstructured object from the event
		objectMap, err := GetObjectMapFromRuntimeObject(&WatchEvent.Object)
		if err != nil {
			logger.Error(err, fmt.Sprintf(runtimeObjectConversionError, err))
			continue
		}

		// Process event for watched object apart
		err = ProcessEvent(ctx, notificationList, objectMap, WatchEvent.Type)
		if err != nil {
			logger.Error(err, fmt.Sprintf(watchedObjectParseError, err))
			continue
		}
	}
}

func ProcessEvent(ctx context.Context, notificationList *[]*jokativ1alpha1.Notification, object map[string]interface{}, eventType watch.EventType) (err error) {
	logger := log.FromContext(ctx)

	if eventType == watch.Added || eventType == watch.Modified || eventType == watch.Deleted {
		for _, notification := range *notificationList {
			var conditionFlags []bool
			for _, condition := range notification.Spec.Conditions {
				parsedKey, err := template.EvaluateTemplate(condition.Key, object)
				if err != nil {
				}
				conditionFlags = append(conditionFlags, parsedKey == condition.Value)
			}

			if slices.Contains(conditionFlags, false) {
				continue
			}

			parsedMessage, err := template.EvaluateTemplate(notification.Spec.Message.Data, object)
			if err != nil {
				// TODO: Update the status of the notification manifest
			}

			// TODO send message
			logger.Info(fmt.Sprintf("Tengo una carta para ti: %s", parsedMessage))

		}
	}
	return err
}
