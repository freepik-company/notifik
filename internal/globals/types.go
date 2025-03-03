package globals

import (
	"context"
	//
	"k8s.io/client-go/dynamic"

	//
	"freepik.com/notifik/api/v1alpha1"
)

// ApplicationT TODO
type applicationT struct {
	// Context TODO
	Context context.Context

	// Configuration TODO
	Configuration v1alpha1.ConfigurationT

	// KubeRawClient TODO
	KubeRawClient *dynamic.DynamicClient
}
