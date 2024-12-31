package metrics

import (
	"context"
)

type Server interface {
	Run(ctx context.Context) error
	Sync(ctx context.Context) error
}
