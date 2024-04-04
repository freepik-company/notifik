package globals

import (
	"context"
	"sync"
)

var (
	Application = applicationT{
		Context: context.Background(),

		WatcherPool: WatcherPoolT{
			Mutex: &sync.Mutex{},
			Pool:  make(map[ResourceTypeName]ResourceTypeWatcherT),
		},
	}
)
