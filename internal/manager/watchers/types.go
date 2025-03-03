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

import "sync"

type ResourceTypeName = string

// ResourceWatcher wraps status and control of a watcher of a resource.
type ResourceWatcher struct {
	mu sync.Mutex

	Started    bool
	StopSignal chan bool
}

// WatchersManager manage watchers' lifecycle
type WatchersManager struct {
	mu       sync.Mutex
	watchers map[ResourceTypeName]*ResourceWatcher
}
