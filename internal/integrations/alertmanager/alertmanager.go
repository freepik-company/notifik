package alertmanager

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func SendMessage(ctx context.Context, reason string, msg string) (err error) {
	logger := log.FromContext(ctx)
	_ = logger

	logger.Info("Hola desde alertmanager")

	return err
}
