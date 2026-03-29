package storage

import (
	"context"

	"lazy-tool/pkg/models"
)

type Store interface {
	UpsertCapability(ctx context.Context, rec models.CapabilityRecord) error
	GetCapability(ctx context.Context, id string) (models.CapabilityRecord, error)
	GetByCanonicalName(ctx context.Context, canonical string) (models.CapabilityRecord, error)
	ListBySource(ctx context.Context, sourceID string) ([]models.CapabilityRecord, error)
	ListAll(ctx context.Context) ([]models.CapabilityRecord, error)
	DeleteStale(ctx context.Context, sourceID string, keepIDs map[string]struct{}) (int, error)
}
