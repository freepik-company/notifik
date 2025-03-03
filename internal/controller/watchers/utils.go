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
	"k8s.io/apimachinery/pkg/runtime"
)

// GetObjectMapFromRuntimeObject converts the runtime.Object to map[string]interface{}
func GetObjectMapFromRuntimeObject(obj *runtime.Object) (objectData map[string]interface{}, err error) {
	objectData, err = runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	return objectData, err
}

// TODO
func GetObjectBasicData(object *map[string]interface{}) (objectData map[string]interface{}, err error) {

	metadata, ok := (*object)["metadata"].(map[string]interface{})
	if !ok {
		err = errors.New("metadata not found or not in expected format")
		return
	}

	objectData = make(map[string]interface{})
	objectData["name"] = metadata["name"]
	objectData["namespace"] = metadata["namespace"]

	return objectData, nil
}
