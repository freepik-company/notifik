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
	"errors"
	"golang.org/x/exp/maps"
	"time"
)

// NewWatchersRegistry TODO
func NewWatchersRegistry() *WatchersRegistry {

	return &WatchersRegistry{
		watchers: make(map[ResourceTypeName]*ResourceWatcher),
	}
}

// RegisterWatcher registers a watcher for required resource type
func (m *WatchersRegistry) RegisterWatcher(rt ResourceTypeName) *ResourceWatcher {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.watchers[rt] = &ResourceWatcher{
		Started:    false,
		StopSignal: make(chan bool),
	}

	return m.watchers[rt]
}

// DisableWatcher send a signal to the watcher to stop
// and delete it from the registry
func (m *WatchersRegistry) DisableWatcher(rt ResourceTypeName) error {
	watcher, exists := m.GetWatcher(rt)
	if !exists {
		return errors.New("watcher not found")
	}

	// Send a signal to stop the watcher
	watcher.mu.Lock()
	watcher.StopSignal <- true
	watcher.mu.Unlock()

	// Wait for some time to stop the watcher
	stoppedWatcher := false
	for i := 0; i < 10; i++ {
		if !m.IsStarted(rt) {
			stoppedWatcher = true
			break
		}
		time.Sleep(1 * time.Second)
	}
	if !stoppedWatcher {
		return errors.New("impossible to stop the watcher")
	}

	// Delete watcher from registry
	m.mu.Lock()
	delete(m.watchers, rt)
	m.mu.Unlock()

	return nil
}

// GetWatcher return the watcher attached to a resource type
func (m *WatchersRegistry) GetWatcher(rt ResourceTypeName) (watcher *ResourceWatcher, exists bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	//
	watcher, exists = m.watchers[rt]
	return watcher, exists
}

// GetRegisteredResourceTypes returns TODO
func (m *WatchersRegistry) GetRegisteredResourceTypes() []ResourceTypeName {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Keys(m.watchers)
}

// SetStarted updates the 'started' flag of a watcher
func (m *WatchersRegistry) SetStarted(rt ResourceTypeName, started bool) error {
	watcher, exists := m.GetWatcher(rt)
	if !exists {
		return errors.New("watcher not found")
	}

	watcher.mu.Lock()
	defer watcher.mu.Unlock()

	watcher.Started = started
	return nil
}

// IsStarted returns whether a watcher of the provided resource type is started or not
func (m *WatchersRegistry) IsStarted(rt ResourceTypeName) bool {
	watcher, exists := m.GetWatcher(rt)
	if !exists {
		return false
	}

	watcher.mu.Lock()
	defer watcher.mu.Unlock()

	return watcher.Started
}
