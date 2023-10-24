package cloudflare

import (
	"context"
	"fmt"
	"github.com/cloudflare/cloudflare-go"
	"github.com/sokdak/dns-ingress/pkg/common"
	"github.com/sokdak/dns-ingress/pkg/environment"
	"github.com/sokdak/dns-ingress/pkg/provider"
	"net/http"
)

const ProviderKey = "cloudflare"

type Client struct {
	CfClient *cloudflare.API
	provider.Client
}

func GenerateCloudFlareClientUsingEnvironment() (*Client, error) {
	if environment.CloudflareAuthKey == nil || environment.CloudflareAuthEmail == nil {
		return nil, fmt.Errorf("can't generate cfclient using envs, missing envs")
	}

	return NewCloudFlareClient(
		*environment.CloudflareAuthKey, *environment.CloudflareAuthEmail, http.DefaultClient,
		*environment.CloudflareClientRateLimit,
		RetryPolicy{
			MaxRetryCount: 0,
			MinDelay:      0,
			MaxDelay:      0,
		}, *environment.CloudflareClientDebugMode)
}

func NewCloudFlareClient(key, email string, client *http.Client, rateLimits float64, retryPolicy RetryPolicy, debug bool) (*Client, error) {
	opts := []cloudflare.Option{
		cloudflare.HTTPClient(client),
		cloudflare.UsingRateLimit(rateLimits),
		cloudflare.UsingRetryPolicy(retryPolicy.MaxRetryCount, retryPolicy.MinDelay, retryPolicy.MaxDelay),
		cloudflare.Debug(debug),
	}
	api, err := cloudflare.New(key, email, opts...)
	if err != nil {
		return nil, fmt.Errorf("can't create new cf-client: %w", err)
	}
	return &Client{
		CfClient: api,
	}, nil
}

func (c *Client) GetZone(ctx context.Context, zoneName string) (*provider.Zone, error) {
	zones, err := c.CfClient.ListZones(ctx, zoneName)
	if err != nil {
		return nil, fmt.Errorf("can't GetZone: %w", err)
	}

	var matchedZones *cloudflare.Zone
	if len(zones) > 0 {
		for _, z := range zones {
			if z.Name == zoneName {
				matchedZones = &z
				break
			}
		}
	}

	if matchedZones == nil {
		return nil, fmt.Errorf("can't GetZone: cannot find zone %s", zoneName)
	}

	return &provider.Zone{
		Id:        matchedZones.ID,
		Name:      matchedZones.Name,
		Activated: !matchedZones.Paused,
	}, nil
}

func (c *Client) GetByName(ctx context.Context, name, zoneId string) (*provider.Domain, error) {
	listParam := cloudflare.ListDNSRecordsParams{
		Name: name,
		ResultInfo: cloudflare.ResultInfo{
			Page:    0,
			PerPage: 0,
		},
	}
	records, _, err := c.CfClient.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(zoneId), listParam)
	if err != nil {
		return nil, fmt.Errorf("can't GetByName: %w", err)
	}

	var r *cloudflare.DNSRecord
	for _, record := range records {
		if record.Name == name {
			r = &record
			break
		}
	}

	if r == nil {
		return nil, fmt.Errorf("can't GetByName: recordset name %s is not found in zoneId %s", name, zoneId)
	}

	return &provider.Domain{
		Id:        r.ID,
		Name:      r.Name,
		Type:      r.Type,
		Records:   []string{r.Content},
		ZoneId:    r.ZoneID,
		ZoneName:  r.ZoneName,
		FQDN:      fmt.Sprintf("%s.%s.", r.Name, r.ZoneName),
		TTL:       r.TTL,
		Activated: true,
	}, nil
}

func (c *Client) Get(ctx context.Context, id, zoneId string) (*provider.Domain, error) {
	r, err := c.CfClient.GetDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneId), id)
	if err != nil {
		return nil, fmt.Errorf("can't Get: %w", err)
	}

	return &provider.Domain{
		Id:        r.ID,
		Name:      r.Name,
		Type:      r.Type,
		Records:   []string{r.Content},
		TTL:       r.TTL,
		ZoneId:    r.ZoneID,
		ZoneName:  r.ZoneName,
		FQDN:      fmt.Sprintf("%s.%s.", r.Name, r.ZoneName),
		Activated: true,
	}, nil
}

func (c *Client) Create(ctx context.Context, name, zoneId, recordType string, records []string, ttl int) (*provider.Domain, error) {
	params := cloudflare.CreateDNSRecordParams{
		Type:    recordType,
		Name:    name,
		Content: records[0],
		TTL:     ttl,
		Proxied: common.BoolPointer(false),
		Locked:  true,
		Comment: "created and managed by dns-ingress.io",
	}
	r, err := c.CfClient.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneId), params)
	if err != nil {
		return nil, fmt.Errorf("can't Create: %w", err)
	}

	return &provider.Domain{
		Id:        r.ID,
		Name:      r.Name,
		Type:      r.Type,
		Records:   []string{r.Content},
		TTL:       r.TTL,
		ZoneId:    r.ZoneID,
		ZoneName:  r.ZoneName,
		FQDN:      fmt.Sprintf("%s.%s.", r.Name, r.ZoneName),
		Activated: true,
	}, nil
}

func (c *Client) Update(ctx context.Context, id, zoneId, recordType string, records []string, ttl int) (*provider.Domain, error) {
	params := cloudflare.UpdateDNSRecordParams{
		ID:      id,
		Type:    recordType,
		Content: records[0],
		TTL:     ttl,
		Proxied: common.BoolPointer(false),
	}
	r, err := c.CfClient.UpdateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneId), params)
	if err != nil {
		return nil, fmt.Errorf("can't Update: %w", err)
	}

	return &provider.Domain{
		Id:        r.ID,
		Name:      r.Name,
		Type:      r.Type,
		Records:   []string{r.Content},
		TTL:       r.TTL,
		ZoneId:    r.ZoneID,
		ZoneName:  r.ZoneName,
		FQDN:      fmt.Sprintf("%s.%s.", r.Name, r.ZoneName),
		Activated: true,
	}, nil
}

func (c *Client) Delete(ctx context.Context, id, zoneId string) error {
	err := c.CfClient.DeleteDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneId), id)
	if err != nil {
		// TODO: consider not found condition
		return fmt.Errorf("can't Delete: %w", err)
	}
	return nil
}
