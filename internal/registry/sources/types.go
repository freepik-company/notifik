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

package sources

import "sync"

type ResourceTypeName = string

// SourcesInformer wraps status and control of an informer of a resource.
type SourcesInformer struct {
	mu sync.Mutex

	Started    bool
	StopSignal chan bool

	// ItemPool represents a pool of stored resources that are being collected by the watcher
	ItemPool []*map[string]any
}

// SourcesRegistry manage sources watchers' lifecycle
type SourcesRegistry struct {
	mu sync.Mutex

	informers map[ResourceTypeName]*SourcesInformer
}
