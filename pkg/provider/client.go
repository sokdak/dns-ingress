package provider

import "context"

type Client interface {
	GetZone(ctx context.Context, zoneName string) (*Zone, error)
	GetByName(ctx context.Context, name, zoneId string) (*Domain, error)
	Get(ctx context.Context, id, zoneId string) (*Domain, error)
	Create(ctx context.Context, name, zoneId, recordType string, records []string, ttl int) (*Domain, error)
	Update(ctx context.Context, id, zoneId, recordType string, records []string, ttl int) (*Domain, error)
	Delete(ctx context.Context, id, zoneId string) error
}
