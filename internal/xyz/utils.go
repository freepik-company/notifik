package xyz

import "k8s.io/apimachinery/pkg/runtime"

// GetObjectMapFromRuntimeObject converts the runtime.Object to map[string]interface{}
func GetObjectMapFromRuntimeObject(obj *runtime.Object) (objectData map[string]interface{}, err error) {
	objectData, err = runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	return objectData, err
}
