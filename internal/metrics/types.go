package metrics

import (
	"context"
	"github.com/quickube/QScaler/api/v1alpha1"
)

type Server interface {
	Run(ctx context.Context) error
	Sync(ctx context.Context) error
	RightSizeContainers(ctx context.Context, worker *v1alpha1.QWorker) error
}
