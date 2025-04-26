package cloudflare

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/cloudflare/cloudflare-go"
)

type Client struct {
	api         *cloudflare.API
	zoneID      string
	domainName  string
	recordName  string
	recordType  string
	recordID    string
	currentURL  string
	currentTTL  int
	currentProxied bool
}

func NewClient(apiToken, zoneID, domainName, recordName string) (*Client, error) {
	api, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloudflare API client: %w", err)
	}

	return &Client{
		api:        api,
		zoneID:     zoneID,
		domainName: domainName,
		recordName: recordName,
		recordType: "CNAME", // Assuming we'll use CNAME for redirects
	}, nil
}

// Initialize gets the current record configuration
func (c *Client) Initialize() error {
	ctx := context.Background()
	
	// Find the record
	records, err := c.api.DNSRecords(ctx, c.zoneID, cloudflare.DNSRecord{
		Name: c.recordName + "." + c.domainName,
		Type: c.recordType,
	})
	
	if err != nil {
		return fmt.Errorf("failed to get DNS records: %w", err)
	}
	
	if len(records) == 0 {
		return errors.New("no matching DNS records found")
	}
	
	// Store the current record details
	record := records[0]
	c.recordID = record.ID
	c.currentURL = record.Content
	c.currentTTL = record.TTL
	c.currentProxied = record.Proxied
	
	log.Printf("Found DNS record: %s -> %s (ID: %s)", record.Name, record.Content, record.ID)
	return nil
}

// UpdateRedirect updates the domain to point to a new URL
func (c *Client) UpdateRedirect(targetURL string) error {
	if targetURL == c.currentURL {
		log.Printf("URL is already set to %s, no update needed", targetURL)
		return nil
	}

	ctx := context.Background()
	
	// Create the updated record
	updatedRecord := cloudflare.DNSRecord{
		Type:    c.recordType,
		Name:    c.recordName,
		Content: targetURL,
		TTL:     c.currentTTL,
		Proxied: c.currentProxied,
	}
	
	// Update the record
	err := c.api.UpdateDNSRecord(ctx, c.zoneID, c.recordID, updatedRecord)
	if err != nil {
		return fmt.Errorf("failed to update DNS record: %w", err)
	}
	
	log.Printf("Successfully updated DNS record to point to: %s", targetURL)
	c.currentURL = targetURL
	return nil
}

// GetCurrentRedirect returns the current redirect URL
func (c *Client) GetCurrentRedirect() string {
	return c.currentURL
}